package functions

var Functions []interface{}

func Register(f interface{}) {
	Functions = append(Functions, f)
}
