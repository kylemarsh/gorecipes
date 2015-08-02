package main

import (
	"fmt"
)

func main() {
	id := 1
	fmt.Println("Loading recipe", id)
	recipe := recipeById(id)
	fmt.Println(recipe)

	fmt.Println("Loading label", id)
	label := labelById(id)
	fmt.Println(label)
}
