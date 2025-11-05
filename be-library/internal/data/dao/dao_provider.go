package dao

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewRecordDAO,
	NewCreditPointDAO,
	NewCommentDAO,
)
