// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
package cbreaker

import (
	"github.com/gin-gonic/gin"
	"github.com/smarty/assertions"
	vegeta "github.com/tsenart/vegeta/lib"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	kgin "github.com/luraproject/lura/v2/router/gin"
)

func setup() {
	go DummyServer()
	go ApiGateway()
	time.Sleep(10 * time.Second)
}

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	os.Exit(retCode)
}

func ApiGateway() {
	logger, err := logging.NewLogger("INFO", os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	parser := config.NewParser()
	serviceConfig, err := parser.Parse("./test.json")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	routerFactory := kgin.DefaultFactory(proxy.NewDefaultFactory(BackendFactory(proxy.HTTPProxyFactory(http.DefaultClient)), logger), logger)
	routerFactory.New().Run(serviceConfig)
}

func DummyServer() {
	r := gin.Default()
	r.GET("/crash", func(c *gin.Context) {
		c.JSON(500, gin.H{
			"message": "boom!",
		})
	})
	_ = r.Run(":8000")
}

func TestCircuitBreaker(t *testing.T) {
	rate := int(4) // per second
	duration := 2 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8080/cbcrash",
	})
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, vegeta.ConstantPacer{Freq: rate, Per: time.Second}, duration, "attacker") {
		metrics.Add(res)
	}
	metrics.Close()
	equal := assertions.ShouldContainKey(metrics.StatusCodes, "500")
	if equal != "" {
		t.Errorf(equal)
	}
}
