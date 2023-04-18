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

func TestUpdate(t *testing.T) {
	UpdateUser()
}

func TestDeleteOne(t *testing.T) {
	DeleteOne()
}

func TestSelectOne(t *testing.T) {
	SelectOne()
}

func TestSelect(t *testing.T) {
	Select()
}

func TestCount(t *testing.T) {
	Count()
}

func TestExec(t *testing.T) {
	Exec()
}

func TestQueryRow(t *testing.T) {
	QueryRow()
}

func TestTransaction(t *testing.T) {
	Transaction()
}
