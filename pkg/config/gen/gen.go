package main

import (
	cfg "github.com/conductorone/baton-demo/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("demo", cfg.Config)
}
