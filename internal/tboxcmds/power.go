package tboxcmds

import (
	"fmt"
	"github.com/mologie/talos-vmtoolsd/internal/nanotoolbox"
	"github.com/sirupsen/logrus"
)

// vmware/guestrpc/powerops.h
const (
	_ = iota
	PowerStateHalt
	PowerStateReboot
	PowerStatePowerOn
	PowerStateResume
	PowerStateSuspend
)

var powerCmdName = map[int]string{
	PowerStateHalt:    "OS_Halt",
	PowerStateReboot:  "OS_Reboot",
	PowerStatePowerOn: "OS_PowerOn",
	PowerStateResume:  "OS_Resume",
	PowerStateSuspend: "OS_Suspend",
}

type PowerDelegate interface {
	Shutdown() error
	Reboot() error
}

type PowerHandler func() error

type powerOp struct {
	log     logrus.FieldLogger
	out     *nanotoolbox.ChannelOut
	state   int
	handler PowerHandler
}

func (op powerOp) Name() string {
	return powerCmdName[op.state]
}

func (op powerOp) HandleCommand([]byte) ([]byte, error) {
	l := op.log.WithField("power_operation", op.Name())
	l.Debug("handling power operation")

	rc := nanotoolbox.RpciOK
	if op.handler != nil {
		if err := op.handler(); err != nil {
			l.WithError(err).Error("error handling power operation")
			rc = nanotoolbox.RpciERR
		}
	}

	msg := fmt.Sprintf("tools.os.statechange.status %s%d\x00", rc, op.state)
	if _, err := op.out.Request([]byte(msg)); err != nil {
		return nil, fmt.Errorf("error sending %q: %w", msg, err)
	}

	return nil, nil
}

func powerOpHandler(svc *nanotoolbox.Service, state int, handler PowerHandler) (string, nanotoolbox.CommandHandler) {
	op := powerOp{
		log:     svc.Log.WithField("command", "power"),
		out:     svc.Out,
		state:   state,
		handler: handler,
	}
	return op.Name(), op.HandleCommand
}

func RegisterPowerDelegate(svc *nanotoolbox.Service, delegate PowerDelegate) {
	svc.AddCapability("tools.capability.statechange")
	svc.RegisterCommandHandler(powerOpHandler(svc, PowerStateHalt, delegate.Shutdown))
	svc.RegisterCommandHandler(powerOpHandler(svc, PowerStateReboot, delegate.Reboot))
	svc.RegisterCommandHandler(powerOpHandler(svc, PowerStatePowerOn, nil))
	svc.RegisterCommandHandler(powerOpHandler(svc, PowerStateSuspend, nil))
	svc.RegisterCommandHandler(powerOpHandler(svc, PowerStateResume, nil))
}
