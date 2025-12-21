package crawler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_GetDetail(t *testing.T) {
	ug, err := NewUnderGrad(
		NewCrawlerClientWithCookieJar(
			10*time.Second,
			NewJarWithCookie(PG_URL, ""),
		),
	)
	if err != nil {
		return
	}

	t.Run("grade", func(t *testing.T) {
		grades, err := ug.GetGrade(context.Background(), 0, 0, 100)
		if err != nil {
			return
		}

		for _, grade := range grades {
			res, err := ug.GetDetail(context.Background(), "2023215153", grade.JX0404ID, grade.CJ0708ID, grade.ZCJ)
			if err != nil {
				return
			}
			fmt.Println(grade)
			fmt.Println(res)
			return
		}

	})
}
