package data

import (
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
)

func ConvertRecordBizToDo(stuID string, list []*biz.Record) []*DO.Record {
	dos := make([]*DO.Record, 0, len(list))
	for _, it := range list {
		dos = append(dos, &DO.Record{
			ID:        it.ID,
			StuID:     stuID,
			SeatID:    it.SeatID,
			RoomID:    it.RoomID,
			RoomName:  it.RoomName,
			BuildName: it.BuildName,
			FloorName: it.FloorName,
			SeatLabel: it.SeatLabel,
			MakeBegin: it.MakeBegin,
			MakeEnd:   it.MakeEnd,
			MakeDate:  it.MakeDate,
			Message:   it.Message,
			Status:    it.Status,
		})
	}
	return dos
}

func ConvertRecordDoToBiz(list []*DO.Record) []*biz.Record {
	records := make([]*biz.Record, 0, len(list))
	for _, li := range list {
		records = append(records, &biz.Record{
			ID:        li.ID,
			RoomName:  li.RoomName,
			RoomID:    li.RoomID,
			BuildName: li.BuildName,
			FloorName: li.FloorName,
			SeatID:    li.SeatID,
			SeatLabel: li.SeatLabel,
			MakeBegin: li.MakeBegin,
			MakeEnd:   li.MakeEnd,
			MakeDate:  li.MakeDate,
			Status:    li.Status,
			Message:   li.Message,
		})
	}
	return records
}

func ConvertDOCreditPointsBiz(summary *DO.CreditSummary, records []DO.CreditRecord) *biz.CreditPoints {
	if summary == nil {
		return &biz.CreditPoints{Summary: nil, Records: nil}
	}
	out := &biz.CreditPoints{
		Summary: &biz.CreditSummary{
			System: summary.System,
			Remain: summary.Remain,
			Total:  summary.Total,
		},
		Records: make([]*biz.CreditRecord, 0, len(records)),
	}
	for _, r := range records {
		out.Records = append(out.Records, &biz.CreditRecord{
			Title:    r.Title,
			Subtitle: r.Subtitle,
			Location: r.Location,
		})
	}
	return out
}

func ConvertBizCreditPointsDO(stuID string, cp *biz.CreditPoints) (*DO.CreditSummary, []DO.CreditRecord) {
	if cp == nil || cp.Summary == nil {
		return nil, nil
	}
	sum := &DO.CreditSummary{
		StuID:  stuID,
		System: cp.Summary.System,
		Remain: cp.Summary.Remain,
		Total:  cp.Summary.Total,
	}
	var recs []DO.CreditRecord
	for _, r := range cp.Records {
		recs = append(recs, DO.CreditRecord{
			StuID:    stuID,
			Title:    r.Title,
			Subtitle: r.Subtitle,
			Location: r.Location,
		})
	}
	return sum, recs
}

//func ConvertDODiscussionBiz(dos []*DO.Discussion) []*biz.Discussion {
//	out := make([]*biz.Discussion, 0, len(dos))
//	for _, d := range dos {
//		item := &biz.Discussion{
//			LabID:    d.LabID,
//			LabName:  d.LabName,
//			KindID:   d.KindID,
//			KindName: d.KindName,
//			DevID:    d.DevID,
//			DevName:  d.DevName,
//			TS:       make([]*biz.DiscussionTS, 0, len(d.TS)),
//		}
//		for _, t := range d.TS {
//			item.TS = append(item.TS, &biz.DiscussionTS{
//				Start:  t.Start,
//				End:    t.End,
//				State:  t.State,
//				Title:  t.Title,
//				Owner:  t.Owner,
//				Occupy: t.Occupy,
//			})
//		}
//		out = append(out, item)
//	}
//	return out
//}
//
//func ConvertBizDiscussionDO(list []*biz.Discussion) []*DO.Discussion {
//	out := make([]*DO.Discussion, 0, len(list))
//	for _, d := range list {
//		item := &DO.Discussion{
//			LabID:    d.LabID,
//			LabName:  d.LabName,
//			KindID:   d.KindID,
//			KindName: d.KindName,
//			DevID:    d.DevID,
//			DevName:  d.DevName,
//			TS:       make([]*DO.DiscussionTS, 0, len(d.TS)),
//		}
//		for _, t := range d.TS {
//			item.TS = append(item.TS, &DO.DiscussionTS{
//				Start:  t.Start,
//				End:    t.End,
//				State:  t.State,
//				Title:  t.Title,
//				Owner:  t.Owner,
//				Occupy: t.Occupy,
//			})
//		}
//		out = append(out, item)
//	}
//	return out
//}
