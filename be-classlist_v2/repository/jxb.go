package repo

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
)

type JxbRepo struct {
	jxbDAO dao.JxbDAO
}

func NewJxbRepo(jxb dao.JxbDAO) JxbRepo {
	return JxbRepo{jxbDAO: jxb}
}

func (j *JxbRepo) SaveJxbSaveJxb(ctx context.Context, stuID string, jxbID []string) error {
	return j.SaveJxbSaveJxb(ctx, stuID, jxbID)
}

func (j *JxbRepo) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	return j.FindStuIdsByJxbId(ctx, jxbId)
}
