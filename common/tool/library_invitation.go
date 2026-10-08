package tool

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
