package util

func PanicIfErr(e error) {
	if e != nil {
		panic(e)
	}
}
