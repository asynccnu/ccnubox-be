package data

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/classService/internal/conf"
	clog "github.com/asynccnu/ccnubox-be/classService/internal/log"
	"github.com/olivere/elastic/v7"
)

func NewEsClient(c *conf.Data) (*elastic.Client, error) {
	ctx := context.Background()

	// 配置 Elasticsearch 的 URL 和嗅探选项
	urlOpt := elastic.SetURL(c.Es.Url)
	sniffOpt := elastic.SetSniff(c.Es.Setsniff)

	// 配置基本认证，使用用户名和密码
	authOpt := elastic.SetBasicAuth(c.Es.Username, c.Es.Password)

	// 创建 Elasticsearch 客户端
	cli, err := elastic.NewClient(urlOpt, sniffOpt, authOpt)
	if err != nil {
		panic(fmt.Sprintf("es connect fail: %v", err))
	}

	clog.LogPrinter.Info("connect to elasticsearch successfully")

	createIndex(ctx, cli, c.Es.KeepDataAfterRestart, classIndexName, classMapping)
	createIndex(ctx, cli, c.Es.KeepDataAfterRestart, freeClassroomIndex, freeClassroomMapping)

	return cli, nil
}

func createIndex(ctx context.Context, cli *elastic.Client, keepData bool, indexName string, mapping string) {
	// 检查索引是否存在
	exist, err := cli.IndexExists(indexName).Do(ctx)
	if err != nil {
		panic(err)
	}
	//如果存在,并且要求保留数据,则返回
	if exist && keepData {
		return
	}
	//下面是不存在或者不保留数据

	// 如果索引存在，先删除索引
	if exist {
		deleteIndex, err := cli.DeleteIndex(indexName).Do(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to delete existing index: %v", err))
		}
		if !deleteIndex.Acknowledged {
			panic("delete index failed")
		}
		clog.LogPrinter.Info("Existing index deleted successfully")
	}

	// 创建新的索引
	createIdx, err := cli.CreateIndex(indexName).BodyString(mapping).Do(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to create index: %v", err))
	}
	if !createIdx.Acknowledged {
		panic("create index failed")
	}
	clog.LogPrinter.Info("Es create index successfully")
}
