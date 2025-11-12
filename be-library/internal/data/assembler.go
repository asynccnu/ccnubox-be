package data

import (
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
)

type Assembler struct{}

func NewAssembler() *Assembler {
	return &Assembler{}
}

func (a *Assembler) ConvertDOCommentBiz(comments []*DO.Comment) []*biz.Comment {
	if len(comments) == 0 {
		return nil
	}
	result := make([]*biz.Comment, 0, len(comments))
	for _, comment := range comments {
		result = append(result, &biz.Comment{
			ID:        comment.ID,
			Floor:     comment.Floor,
			SeatID:    comment.SeatID,
			Username:  comment.Username,
			Content:   comment.Content,
			Rating:    comment.Rating,
			CreatedAt: comment.CreatedAt,
		})
	}
	return result
}
