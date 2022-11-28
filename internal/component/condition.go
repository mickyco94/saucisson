package component

// Condition is an abstraction that represents a
// particular condition being satisified. When the condition
// has been satisfied a notification is pushed via the registered channel
//
// Examples of conditions:
// - File being created
// - CRON schedule being triggered
// - Process activating
type Condition interface {

	// Register takes a channel to notify satisfying the condition
	// The channel is pushed to once the condition has been satisfied
	// An error is returned if the registration of the condition fails

	//TODO: Re-evaluate this interface, it's the only reason we need
	//component package to have knowledge of app dependencies
	//The methods don't really do anything themselves as well...
	//Maybe there's a way to invert it
	//Services register conditions, instead of the other way around
	Register(func()) error
}
