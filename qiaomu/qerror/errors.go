package qerror

type QError struct {
	err    error
	ErrFuc ErrorFuc
}

func Default() *QError {
	return &QError{}
}
func (e *QError) Error() string {
	return e.err.Error()
}

func (e *QError) Put(err error) {
	e.check(err)
}

func (e *QError) check(err error) {
	if err != nil {
		e.err = err
		panic(e)
	}
}

type ErrorFuc func(msError *QError)

//暴露一个方法 让用户自定义

func (e *QError) Result(errFuc ErrorFuc) {
	e.ErrFuc = errFuc
}
func (e *QError) ExecResult() {
	e.ErrFuc(e)
}
