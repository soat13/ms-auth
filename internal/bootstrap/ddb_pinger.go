package bootstrap

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DDBPinger struct {
	client *dynamodb.Client
	table  string
}

func NewDDBPinger(c *dynamodb.Client, table string) *DDBPinger {
	return &DDBPinger{client: c, table: table}
}

func (p *DDBPinger) PingContext(ctx context.Context) error {
	_, err := p.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(p.table),
	})
	return err
}
