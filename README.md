Conditions:

- Processes being started
- Files being created
- CRON

Execute:

- Shell
- HTTP
- IO (e.g. create file)
- ...

//Biggest question atm is how to attach conditions to those services that evaluate conditions. Also how do we have the conditions trigger executions.
//Some kind of package like HTTP that generically attachs funcs to conditions might make sense here.

//There are ambient underlying services, similar to HTTP. That have funcs attached to them. They also need to be configured, e.g. filelistener

//This interface would be great!!
//If Condition also had a name then that would be great
`thing.HandleFunc(component.Condition, component.Executor)`
`thing.HandleCron(component.Cron, component.Executor)`
`thing.HandleFile(component.File, component.Executor)`
