package main

import (
	"github.com/joho/godotenv"
	"microservice_swd_demo/cmd"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	cmd.Execute()
}
