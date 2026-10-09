package tool

import (
	"errors"
	"sort"
)

// IsValidLibraryTeamID 校验学校队伍 ID 是否为 1～128 字节的 ASCII 十进制数字字符串。
// 不接受空串、空白、符号或非 ASCII 数字，不限制数值范围。
// 例如：IsValidLibraryTeamID("2102744440918429696") 返回 true，IsValidLibraryTeamID("") 返回 false。
func IsValidLibraryTeamID(teamID string) bool {
	if len(teamID) == 0 || len(teamID) > 128 {
		return false
	}
	for _, ch := range teamID {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// NormalizeTeamInvitation 校验发起人、队伍 ID 和收件人，并返回去重、排序后的收件人。
// 先按 maxRecipients 和 20 人硬上限检查原始人数，拒绝自通知，避免重复项绕过批量限制。
func NormalizeTeamInvitation(operator, teamID string, recipients []string, maxRecipients int) ([]string, error) {
	if !IsValidStudentID(operator) || !IsValidLibraryTeamID(teamID) || len(recipients) == 0 || len(recipients) > maxRecipients || len(recipients) > 20 {
		return nil, errors.New("invalid team invitation parameters")
	}
	unique := make(map[string]struct{}, len(recipients))
	for _, recipient := range recipients {
		if !IsValidStudentID(recipient) || recipient == operator {
			return nil, errors.New("invalid invitation recipient")
		}
		unique[recipient] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for recipient := range unique {
		result = append(result, recipient)
	}
	sort.Strings(result)
	return result, nil
}
