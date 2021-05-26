package host

import (
	"github.com/filanov/stateswitch"
	"github.com/openshift/assisted-service/models"
)

func NewPoolClusterHostStateMachine(th *transitionHandler) stateswitch.StateMachine {
	sm := stateswitch.NewStateMachine()

	// Register host
	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeRegisterHost,
		SourceStates: []stateswitch.State{
			"",
			stateswitch.State(models.HostStatusDiscovering),
			stateswitch.State(models.HostStatusDisconnected),
			stateswitch.State(models.HostStatusInsufficient),
			stateswitch.State(models.HostStatusReadyToBeMoved),
		},
		DestinationState: stateswitch.State(models.HostStatusDiscovering),
		PostTransition:   th.PostRegisterHost,
	})

	// Disabled host can register if it was booted, no change in the state.
	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType:   TransitionTypeRegisterHost,
		SourceStates:     []stateswitch.State{stateswitch.State(models.HostStatusDisabled)},
		DestinationState: stateswitch.State(models.HostStatusDisabled),
	})

	// Disable host
	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeDisableHost,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDisconnected),
			stateswitch.State(models.HostStatusDiscovering),
			stateswitch.State(models.HostStatusInsufficient),
			stateswitch.State(models.HostStatusReadyToBeMoved),
		},
		DestinationState: stateswitch.State(models.HostStatusDisabled),
		PostTransition:   th.PostDisableHost,
	})

	// Enable host
	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeEnableHost,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDisabled),
		},
		DestinationState: stateswitch.State(models.HostStatusDiscovering),
		PostTransition:   th.PostEnableHost,
	})

	// Refresh host

	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeRefresh,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDiscovering),
			stateswitch.State(models.HostStatusInsufficient),
			stateswitch.State(models.HostStatusReadyToBeMoved),
			stateswitch.State(models.HostStatusDisconnected),
		},
		Condition:        stateswitch.Not(If(IsConnected)),
		DestinationState: stateswitch.State(models.HostStatusDisconnected),
		PostTransition:   th.PostRefreshHost(statusInfoDisconnected),
	})

	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeRefresh,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDisconnected),
			stateswitch.State(models.HostStatusDiscovering),
		},
		Condition:        stateswitch.And(If(IsConnected), stateswitch.Not(If(HasInventory))),
		DestinationState: stateswitch.State(models.HostStatusDiscovering),
		PostTransition:   th.PostRefreshHost(statusInfoDiscovering),
	})

	var hasMinRequiredHardware = stateswitch.And(If(HasMinValidDisks), If(HasMinCPUCores), If(HasMinMemory), If(IsPlatformValid))

	// In order for this transition to be fired at least one of the validations in minRequiredHardwareValidations must fail.
	// This transition handles the case that a host does not pass minimum hardware requirements for any of the roles
	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeRefresh,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDisconnected),
			stateswitch.State(models.HostStatusDiscovering),
			stateswitch.State(models.HostStatusInsufficient),
			stateswitch.State(models.HostStatusReadyToBeMoved),
		},
		Condition: stateswitch.And(If(IsConnected), If(HasInventory),
			stateswitch.Not(hasMinRequiredHardware)),
		DestinationState: stateswitch.State(models.HostStatusInsufficient),
		PostTransition:   th.PostRefreshHost(statusInfoInsufficientHardware),
	})

	// Noop transitions
	for _, state := range []stateswitch.State{
		stateswitch.State(models.HostStatusDisabled),
	} {
		sm.AddTransition(stateswitch.TransitionRule{
			TransitionType:   TransitionTypeRefresh,
			SourceStates:     []stateswitch.State{state},
			DestinationState: state,
		})
	}

	sm.AddTransition(stateswitch.TransitionRule{
		TransitionType: TransitionTypeRefresh,
		SourceStates: []stateswitch.State{
			stateswitch.State(models.HostStatusDisconnected),
			stateswitch.State(models.HostStatusDiscovering),
			stateswitch.State(models.HostStatusInsufficient),
			stateswitch.State(models.HostStatusReadyToBeMoved),
		},
		Condition: stateswitch.And(If(IsConnected), If(HasInventory),
			hasMinRequiredHardware),
		DestinationState: stateswitch.State(models.HostStatusReadyToBeMoved),
		PostTransition:   th.PostRefreshHost(statusInfoHostReadyToBeMoved),
	})

	return sm
}
