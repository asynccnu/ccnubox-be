package dao

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GradeDAO interface {
	FindGrades(ctx context.Context, studentID string, xnm, xqm int64) ([]model.Grade, error)
	SyncGrades(ctx context.Context, snapshot GradeSnapshot) (GradeSyncResult, error)
	UpdateGradeDetails(ctx context.Context, grades []model.Grade) error
	GetDistinctGradeType(ctx context.Context, studentID string) ([]string, error)
}

type GradeSnapshot struct {
	StudentID             string
	SourceSystem          string
	Grades                []model.Grade
	Unscored              []model.Grade // 只表示课程仍在上游，不能写入成绩或通知
	Complete              bool
	ObservedAt            time.Time
	SuppressNotifications bool
}

// Notify 仅供本轮变更反馈和同班提速，通知发布只能读取 Outbox。
type GradeSyncResult struct {
	Changed []model.Grade
	Notify  []model.Grade
	Skipped []model.Grade // 身份不明的课程单独保留待核验，不阻断其他课程
}

type gradeKey struct {
	StudentID string
	JxbID     string
}

// 单独保留旧总分的 NULL 信息，不能把历史缺失成绩读成有效零分。
type storedGrade struct {
	model.Grade
	HasTotalScore bool
}

type gradeDAO struct{ db *gorm.DB }

func NewGradeDAO(db *gorm.DB) GradeDAO { return &gradeDAO{db: db} }

const maxGradeTransactionAttempts = 5

var (
	ErrGradeVersionChanged = errors.New("grade changed while fetching details")
	ErrGradeIdentity       = errors.New("grade identity requires verification")
)

func (d *gradeDAO) FindGrades(ctx context.Context, studentID string, xnm, xqm int64) ([]model.Grade, error) {
	var grades []model.Grade
	query := d.db.WithContext(ctx).Where("student_id = ?", studentID)
	if xnm != 0 {
		query = query.Where("xnm = ?", xnm)
	}
	if xqm != 0 {
		query = query.Where("xqm = ?", xqm)
	}
	err := query.Find(&grades).Error
	return grades, err
}

func (d *gradeDAO) SyncGrades(ctx context.Context, snapshot GradeSnapshot) (GradeSyncResult, error) {
	if snapshot.ObservedAt.Unix() <= 0 || strings.TrimSpace(snapshot.StudentID) == "" || snapshot.SourceSystem == "" {
		return GradeSyncResult{}, errorx.New("dao: invalid grade snapshot")
	}
	for _, batch := range [][]model.Grade{snapshot.Grades, snapshot.Unscored} {
		for _, grade := range batch {
			if grade.StudentId != snapshot.StudentID {
				return GradeSyncResult{}, errorx.New("dao: snapshot student mismatch")
			}
		}
	}
	grades, skipped, err := prepareGradeSnapshot(snapshot.Grades)
	if err != nil {
		return GradeSyncResult{}, d.gradeSyncError(ctx, snapshot, err)
	}
	var result GradeSyncResult
	err = d.gradeTransaction(ctx, func(tx *gorm.DB) error {
		result = GradeSyncResult{Skipped: append([]model.Grade(nil), skipped...)}
		state, created, err := lockGradeSyncState(tx, snapshot.StudentID)
		if err != nil {
			return err
		}
		var existing []storedGrade
		if err := tx.Model(&model.Grade{}).Select("*, cj IS NOT NULL AS has_total_score").
			Clauses(clause.Locking{Strength: "UPDATE"}).Where("student_id = ?", snapshot.StudentID).Find(&existing).Error; err != nil {
			return err
		}
		// 只在本事务新建状态时继承旧基线，不能把首次导入后留下的未初始化状态误认成老用户。
		inheritedBaseline := created && len(existing) > 0
		notifyEnabled := !snapshot.SuppressNotifications && (state.InitializedAt != 0 || inheritedBaseline)
		byClass, byIdentity := map[string]storedGrade{}, map[string]storedGrade{}
		for _, grade := range existing {
			byClass[grade.JxbId] = grade
			if grade.NotificationID != nil && grade.ValidNotificationState() {
				byIdentity[grade.NotificationIdentity()] = grade
			}
		}
		var aliases []model.GradeIdentityAlias
		if err := tx.Where("student_id = ? AND source_system = ? AND alias_kind = ?", snapshot.StudentID, snapshot.SourceSystem, "jxb_id").Find(&aliases).Error; err != nil {
			return err
		}
		byAlias := map[string]string{}
		for _, alias := range aliases {
			byAlias[alias.AliasValue] = alias.NotificationID
		}
		unmatchedLegacy := hasUnconfirmedLegacyGrade(existing, grades, skipped, snapshot.Unscored, byAlias)
		seen := map[string]bool{}
		for _, grade := range grades {
			old, exists := byClass[grade.JxbId]
			aliasID, confirmed := byAlias[grade.JxbId]
			if confirmed {
				mapped, ok := byIdentity[aliasID]
				if !ok || (exists && old.NotificationIdentity() != aliasID) {
					return ErrGradeIdentity
				}
				old, exists = mapped, true
			}
			if exists && !old.ValidNotificationState() {
				return ErrGradeIdentity
			}
			scoreChanged := !exists || !old.HasTotalScore || old.Cj != grade.Cj
			storageChanged := !exists || !old.HasTotalScore || old.JxbId != grade.JxbId || !isGradeEqual(old.Grade, grade)
			trackScore := confirmed || !isPseudoGradeID(grade)
			if !confirmed && exists && isPseudoGradeID(old.Grade) &&
				(old.NotificationID != nil || old.Kcmc != grade.Kcmc || old.Xnm != grade.Xnm || old.Xqm != grade.Xqm) {
				result.Skipped = append(result.Skipped, grade)
				continue
			}
			if !exists && !confirmed && unmatchedLegacy {
				// 不能把未对应的旧伪 ID 重建成新课；仅隔离这门课，已确认课程继续同步。
				result.Skipped = append(result.Skipped, grade)
				continue
			}
			reason := "first_published"
			if exists {
				grade.NotificationID, grade.ScoreVersion = old.NotificationID, old.ScoreVersion
				grade.ChangeVersion = old.ChangeVersion
				grade.RegularGradePercent, grade.RegularGrade = old.RegularGradePercent, old.RegularGrade
				grade.FinalGradePercent, grade.FinalGrade = old.FinalGradePercent, old.FinalGrade
				if storageChanged {
					grade.ChangeVersion++
				}
				if scoreChanged && trackScore {
					grade.ScoreVersion++
					if old.HasTotalScore {
						reason = "score_changed"
					}
				}
			} else {
				grade.ChangeVersion, grade.ScoreVersion = 1, 0
				grade.NotificationID = nil
			}
			if trackScore && (!exists || scoreChanged) && grade.NotificationID == nil {
				id, err := model.NewNotificationID()
				if err != nil {
					return err
				}
				grade.NotificationID, grade.ScoreVersion = &id, 1
			}
			if !grade.ValidNotificationState() {
				return ErrGradeIdentity
			}
			if grade.NotificationID != nil {
				id := grade.NotificationIdentity()
				if seen[id] {
					return ErrGradeIdentity
				}
				seen[id] = true
			}
			if storageChanged {
				if exists {
					if err := tx.Model(&model.Grade{}).Where("student_id = ? AND jxb_id = ?", old.StudentId, old.JxbId).
						Select(gradeUpdateColumns(false)).Updates(&grade).Error; err != nil {
						return err
					}
				} else if err := tx.Create(&grade).Error; err != nil {
					return err
				}
				result.Changed = append(result.Changed, grade)
			}
			if grade.NotificationID != nil && !confirmed && storageChanged && !isPseudoGradeID(grade) {
				alias := model.GradeIdentityAlias{
					StudentID: snapshot.StudentID, SourceSystem: snapshot.SourceSystem,
					AliasKind: "jxb_id", AliasValue: grade.JxbId, NotificationID: grade.NotificationIdentity(),
				}
				if err := tx.Create(&alias).Error; err != nil {
					return err
				}
			}
			if notifyEnabled && trackScore && scoreChanged {
				event, err := domain.NewGradeNotification(grade, reason, snapshot.ObservedAt.Unix())
				if err != nil {
					return err
				}
				if err := tx.Create(&event).Error; err != nil {
					return err
				}
				result.Notify = append(result.Notify, grade)
			}
		}
		if len(result.Skipped) > 0 {
			if err := saveGradeSyncIssue(tx, snapshot, ErrGradeIdentity); err != nil {
				return err
			}
		}
		// 部分导入先留下未初始化状态，后续完整同步不能把它误认成历史基线。
		if !snapshot.SuppressNotifications && state.InitializedAt == 0 && (inheritedBaseline || snapshot.Complete) {
			state.InitializedAt = snapshot.ObservedAt.Unix()
		}
		state.LastSuccessAt = snapshot.ObservedAt.Unix()
		return tx.Save(&state).Error
	})
	if err != nil {
		return GradeSyncResult{}, d.gradeSyncError(ctx, snapshot, err)
	}
	return result, nil
}

// 缺失时插入状态，并发冲突由外层重试；不依赖 INSERT 的 RowsAffected 判断首次创建。
func lockGradeSyncState(tx *gorm.DB, studentID string) (model.GradeSyncState, bool, error) {
	var state model.GradeSyncState
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("student_id = ?", studentID).Limit(1).Find(&state)
	if query.Error != nil || query.RowsAffected > 0 {
		return state, false, query.Error
	}
	state.StudentID = studentID
	return state, true, tx.Create(&state).Error
}

func hasUnconfirmedLegacyGrade(existing []storedGrade, grades, skipped, unscored []model.Grade, aliases map[string]string) bool {
	present := map[string]bool{}
	matched := map[string]bool{}
	// 冲突或暂未发布成绩的旧课仍在快照中，不能因单独跳过它而阻断其他新课。
	for _, batch := range [][]model.Grade{grades, skipped, unscored} {
		for _, grade := range batch {
			key := grade.LocalJxbID()
			if key == "" {
				continue
			}
			present[key] = true
			if id, ok := aliases[key]; ok {
				matched[id] = true
			}
		}
	}
	for _, old := range existing {
		if !isPseudoGradeID(old.Grade) || present[old.JxbId] {
			continue
		}
		if id := old.NotificationIdentity(); old.NotificationID != nil && old.ValidNotificationState() && (matched[id] || aliases[old.JxbId] == id) {
			continue
		}
		return true
	}
	return false
}

func (d *gradeDAO) gradeSyncError(ctx context.Context, snapshot GradeSnapshot, err error) error {
	if !errors.Is(err, ErrGradeIdentity) {
		return err
	}
	if saveErr := saveGradeSyncIssue(d.db.WithContext(ctx), snapshot, err); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	return err
}

func saveGradeSyncIssue(db *gorm.DB, snapshot GradeSnapshot, cause error) error {
	// 原始快照只包含结构化成绩，不含 Cookie、Token 或教务原始响应。
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	issue := model.GradeSyncIssue{StudentID: snapshot.StudentID, Payload: string(payload), LastError: cause.Error()}
	return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&issue).Error
}

func (d *gradeDAO) UpdateGradeDetails(ctx context.Context, grades []model.Grade) error {
	grades, err := prepareGradeBatch(grades)
	if err != nil || len(grades) == 0 {
		return err
	}
	return d.gradeTransaction(ctx, func(tx *gorm.DB) error {
		for _, grade := range grades {
			var old model.Grade
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, "student_id = ? AND jxb_id = ?", grade.StudentId, grade.JxbId).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if gradeDetailsEqual(old, grade) {
				continue
			}
			if old.ChangeVersion != grade.ChangeVersion {
				return ErrGradeVersionChanged
			}
			grade.ChangeVersion++
			if err := tx.Model(&model.Grade{}).Where("student_id = ? AND jxb_id = ?", grade.StudentId, grade.JxbId).
				Select(gradeUpdateColumns(true)).Updates(&grade).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *gradeDAO) gradeTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
	var err error
	for attempt := 1; attempt <= maxGradeTransactionAttempts; attempt++ {
		err = d.db.WithContext(ctx).Transaction(fn)
		if err == nil || ctx.Err() != nil || !isRetryableGradeTransactionError(err) || attempt == maxGradeTransactionAttempts {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt) * 10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

// 空教学班沿用旧本地键，但不赋予通知身份；同名同学期的冲突记录必须整组隔离。
func prepareGradeSnapshot(grades []model.Grade) ([]model.Grade, []model.Grade, error) {
	normalized := make([]model.Grade, 0, len(grades))
	var skipped []model.Grade
	seen := map[gradeKey]model.Grade{}
	ambiguous := map[gradeKey]bool{}
	for _, grade := range grades {
		if strings.TrimSpace(grade.StudentId) == "" || math.IsNaN(float64(grade.Cj)) || math.IsInf(float64(grade.Cj), 0) {
			return nil, nil, errorx.New("dao: invalid grade student or total score")
		}
		grade.JxbId = grade.LocalJxbID()
		if grade.JxbId == "" {
			skipped = append(skipped, grade)
			continue
		}
		key := gradeKey{StudentID: grade.StudentId, JxbID: grade.JxbId}
		if previous, ok := seen[key]; ok && isPseudoGradeID(grade) &&
			(!isGradeEqual(previous, grade) || !gradeDetailsEqual(previous, grade) || previous.ChangeVersion != grade.ChangeVersion) {
			ambiguous[key] = true
		}
		seen[key] = grade
		normalized = append(normalized, grade)
	}
	candidates := make([]model.Grade, 0, len(normalized))
	for _, grade := range normalized {
		if ambiguous[gradeKey{StudentID: grade.StudentId, JxbID: grade.JxbId}] {
			skipped = append(skipped, grade)
			continue
		}
		candidates = append(candidates, grade)
	}
	prepared, err := prepareGradeBatch(candidates)
	return prepared, skipped, err
}

func prepareGradeBatch(grades []model.Grade) ([]model.Grade, error) {
	result := make([]model.Grade, 0, len(grades))
	seen := map[gradeKey]model.Grade{}
	for _, grade := range grades {
		if strings.TrimSpace(grade.StudentId) == "" {
			return nil, errorx.New("dao: grade student is empty")
		}
		if strings.TrimSpace(grade.JxbId) == "" {
			return nil, ErrGradeIdentity
		}
		if math.IsNaN(float64(grade.Cj)) || math.IsInf(float64(grade.Cj), 0) {
			return nil, errorx.New("dao: invalid total score")
		}
		key := gradeKey{StudentID: grade.StudentId, JxbID: grade.JxbId}
		if previous, ok := seen[key]; ok {
			if !isGradeEqual(previous, grade) || !gradeDetailsEqual(previous, grade) || previous.ChangeVersion != grade.ChangeVersion {
				return nil, errorx.New("dao: conflicting grades in snapshot")
			}
			continue
		}
		seen[key] = grade
		result = append(result, grade)
	}
	return result, nil
}

func isRetryableGradeTransactionError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062, 1205, 1213:
			return true
		}
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked")
}

func gradeUpdateColumns(detail bool) []string {
	if detail {
		return []string{"regular_grade_percent", "regular_grade", "final_grade_percent", "final_grade", "change_version"}
	}
	return []string{"jxb_id", "kc_id", "kcmc", "xnm", "xqm", "xf", "kcxzmc", "kclbmc", "kcbj", "jd", "cj", "change_version", "notification_id", "score_version"}
}

func isPseudoGradeID(g model.Grade) bool {
	return !g.HasRealJxbID()
}

func gradeDetailsEqual(a, b model.Grade) bool {
	return a.RegularGradePercent == b.RegularGradePercent && a.RegularGrade == b.RegularGrade &&
		a.FinalGradePercent == b.FinalGradePercent && a.FinalGrade == b.FinalGrade
}

func isGradeEqual(a, b model.Grade) bool {
	return a.Kcmc == b.Kcmc && a.KcId == b.KcId && a.Xnm == b.Xnm && a.Xqm == b.Xqm && a.Xf == b.Xf &&
		a.Kcxzmc == b.Kcxzmc && a.Kclbmc == b.Kclbmc && a.Kcbj == b.Kcbj && a.Jd == b.Jd && a.Cj == b.Cj
}

func (d *gradeDAO) GetDistinctGradeType(ctx context.Context, studentID string) ([]string, error) {
	var results []string
	err := d.db.WithContext(ctx).Model(&model.Grade{}).Where("student_id = ?", studentID).Distinct("kcxzmc").Pluck("kcxzmc", &results).Error
	return results, err
}
