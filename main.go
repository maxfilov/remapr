package main

import (
	"context"
	_ "embed"
	"errors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed configuration/routes.yaml
var configurationData []byte

type Route struct {
	Transform string    `yaml:"transform"`
	Backend   *YamlURL  `yaml:"backend"`
	Rewrite   *YamlPath `yaml:"rewrite"`
}

type Configuration struct {
	Routes map[string]Route `yaml:"routes"`
}

func main() {
	loggerConfig := makeLoggerConfig()
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}
	logger.Info("started application")

	engine, err := makeEngine(*loggerConfig)
	if err != nil {
		panic(err)
	}

	var configuration Configuration
	err = yaml.Unmarshal(configurationData, &configuration)
	if err != nil {
		panic(err)
	}
	for path, route := range configuration.Routes {
		logger := logger.With(zap.String("path", path))
		logger.Info("setting up the path")
		proxy, err := makeHandler(path, route)
		if err != nil {
			panic(err)
		}
		engine.POST(path, gin.WrapH(proxy))
	}
	listener, err := net.Listen("tcp", ":8080")
	server := &http.Server{
		Handler: engine,
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	retCode := 0
	sigCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case sig := <-signalChan:
			if unixSignal, ok := sig.(syscall.Signal); ok {
				retCode = 128 + int(unixSignal)
			}
			shutdownError := server.Shutdown(context.Background())
			if shutdownError != nil {
				logger.Error("can not shutdown server", zap.Error(shutdownError))
				closeError := server.Close()
				if closeError != nil {
					logger.Error("can not close server", zap.Error(closeError))
				}
			}
		case <-sigCtx.Done():
			return
		}
		signal.Ignore(os.Interrupt)
		logger.Info("stopped listening for signals")
		close(signalChan)
		logger.Info("closed signal channel")
	}()
	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
	os.Exit(retCode)
}

func makeLoggerConfig() *zap.Config {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.TimeKey = "date"
	return &config
}

func makeHandler(path string, route Route) (*ProxyHandler, error) {
	var transform JsonTransform
	transform, err := NewJQJsonTransform(route.Transform)
	if err != nil {
		return nil, err
	}
	proxy := ProxyHandler{
		method:    http.MethodGet,
		path:      path,
		transform: transform,
	}
	if route.Rewrite != nil {
		proxy.path = route.Rewrite.Path
	}
	return &proxy, nil
}

func makeEngine(logConfig zap.Config) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	logConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	ginLogger, err := logConfig.Build()
	if err != nil {
		return nil, err
	}
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(gin.Recovery())
	engine.Use(ginzap.Ginzap(ginLogger, time.RFC3339, false))
	engine.RedirectTrailingSlash = true
	return engine, nil
}
