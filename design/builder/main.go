package main

import (
	"daydayup/design/builder/module"
	"fmt"
)

// 当所需产品较为复杂且需要多个步骤才能完成时， 也可以使用生成器模式。
//在这种情况下， 使用多个构造方法比仅仅使用一个复杂可怕的构造函数更简单。
//分为多个步骤进行构建的潜在问题是， 构建不完整的和不稳定的产品可能会被暴露给客户端。 生成器模式能够在产品完成构建之前使其处于私密状态。
// 在下方的代码中， 我们可以看到 igloo­Builder冰屋生成器与 normal­Builder普通房屋生成器可建造不同类型房屋， 即 igloo冰屋和 normal­House普通房屋 。 每种房屋类型的建造步骤都是相同的。 主管 （可选） 结构体可对建造过程进行组织。

func main() {
	normalBuilder := module.GetBuilder("normal")
	director := module.NewDirector(normalBuilder)
	normalHouse := director.BuildHouse()
	fmt.Printf("Normal House Door Type: %v\n", normalHouse)

	iglooBuilder := module.GetBuilder("igloo")
	director.SetBuilder(iglooBuilder)
	iglooHouse := director.BuildHouse()
	fmt.Printf("Normal House Door Type: %v\n", iglooHouse)

}
