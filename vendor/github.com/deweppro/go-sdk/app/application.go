package app

import (
	"os"

	"github.com/deweppro/go-sdk/console"
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/syscall"
)

type (
	//ENV type for enviremants (prod, dev, stage, etc)
	ENV string

	App interface {
		Logger(log log.Logger) App
		Modules(modules ...interface{}) App
		ConfigFile(filename string, configs ...interface{}) App
		Run()
		Invoke(call interface{})
	}

	_app struct {
		cfile    string
		configs  Modules
		modules  Modules
		sources  Sources
		packages *_dic
		logout   *_log
		log      log.Logger
		ctx      Context
	}
)

// New create application
func New() App {
	ctx := NewContext()
	return &_app{
		modules:  Modules{},
		configs:  Modules{},
		packages: newDic(ctx),
		ctx:      ctx,
	}
}

// Logger setup logger
func (a *_app) Logger(log log.Logger) App {
	a.log = log
	return a
}

// Modules append object to modules list
func (a *_app) Modules(modules ...interface{}) App {
	for _, mod := range modules {
		switch v := mod.(type) {
		case Modules:
			a.modules = a.modules.Add(v...)
		default:
			a.modules = a.modules.Add(v)
		}
	}

	return a
}

// ConfigFile set config file path and configs models
func (a *_app) ConfigFile(filename string, configs ...interface{}) App {
	a.cfile = filename
	for _, config := range configs {
		a.configs = a.configs.Add(config)
	}

	return a
}

// Run run application
func (a *_app) Run() {
	a.prepareConfig(false)

	result := a.steps(
		[]step{
			{
				Message: "Registering dependencies",
				Call:    func() error { return a.packages.Register(a.modules...) },
			},
			{
				Message: "Running dependencies",
				Call:    func() error { return a.packages.Build() },
			},
		},
		func(er bool) {
			if er {
				a.ctx.Close()
				return
			}
			go syscall.OnStop(a.ctx.Close)
			<-a.ctx.Done()
		},
		[]step{
			{
				Message: "Stop dependencies",
				Call:    func() error { return a.packages.Down() },
			},
		},
	)
	console.FatalIfErr(a.logout.Close(), "close log file")
	if result {
		os.Exit(1)
	}
	os.Exit(0)
}

// Invoke run application
func (a *_app) Invoke(call interface{}) {
	a.prepareConfig(true)

	result := a.steps(
		[]step{
			{
				Message: "Registering dependencies",
				Call:    func() error { return a.packages.Register(a.modules...) },
			},
			{
				Message: "Running dependencies",
				Call:    func() error { return a.packages.Invoke(call) },
			},
		},
		func(_ bool) {},
		[]step{
			{
				Message: "Stop dependencies",
				Call:    func() error { return a.packages.Down() },
			},
		},
	)
	console.FatalIfErr(a.logout.Close(), "close log file")
	if result {
		os.Exit(1)
	}
	os.Exit(0)
}

func (a *_app) prepareConfig(interactive bool) {
	var err error
	if len(a.cfile) == 0 {
		a.logout = newLog(&Config{
			Level:   4,
			LogFile: "/dev/stdout",
		})
		a.log = log.Default()
		a.logout.Handler(a.log)
	}
	if len(a.cfile) > 0 {
		// read config file
		a.sources = Sources(a.cfile)

		// init logger
		config := &Config{}
		if err = a.sources.Decode(config); err != nil {
			console.FatalIfErr(err, "decode config file: %s", a.cfile)
		}
		if interactive {
			config.Level = 4
			config.LogFile = "/dev/stdout"
		}
		a.logout = newLog(config)
		if a.log == nil {
			a.log = log.Default()
		}
		a.logout.Handler(a.log)
		a.modules = a.modules.Add(func() log.Logger { return a.log }, ENV(config.Env))

		// decode all configs
		var configs []interface{}
		configs, err = typingRefPtr(a.configs, func(i interface{}) error {
			return a.sources.Decode(i)
		})
		if err != nil {
			a.log.WithFields(log.Fields{
				"err": err.Error(),
			}).Fatalf("Decode config file")
		}
		a.modules = a.modules.Add(configs...)

		if !interactive && len(config.PidFile) > 0 {
			if err = syscall.Pid(config.PidFile); err != nil {
				a.log.WithFields(log.Fields{
					"err":  err.Error(),
					"file": config.PidFile,
				}).Fatalf("Create pid file")
			}
		}
	}
}

type step struct {
	Call    func() error
	Message string
}

func (a *_app) steps(up []step, wait func(bool), down []step) bool {
	var erc int

	for _, s := range up {
		a.log.Infof(s.Message)
		if err := s.Call(); err != nil {
			a.log.WithFields(log.Fields{
				"err": err.Error(),
			}).Errorf(s.Message)
			erc++
			break
		}
	}

	wait(erc > 0)

	for _, s := range down {
		a.log.Infof(s.Message)
		if err := s.Call(); err != nil {
			a.log.WithFields(log.Fields{
				"err": err.Error(),
			}).Errorf(s.Message)
			erc++
		}
	}

	return erc > 0
}
