package main

func main() {

}

type A struct {
	b bool
}
func init() {
	b := []bool{}
	a := A{b: true}
	b = append(b, a.b)
	println(b[0])
}