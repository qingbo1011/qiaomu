package service

import (
	"fmt"
	"testing"

	"github.com/qingbo1011/qiaomu/orm"
)

func TestName(t *testing.T) {
	fmt.Println(orm.Name("UserNameTestHaaLooH"))
}

func TestSaveUser(t *testing.T) {
	SaveUser()
}

func TestSaveUserBatch(t *testing.T) {
	SaveUserBatch()
}
