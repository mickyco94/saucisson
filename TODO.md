# TODO

[x] Improve config parsing
[x] Refactor condition factory + executor factory
[ ] Allow Parallel option, should services iterations be duplicated
[] Spawn vs. Exec, for synchronous shell commands. Refactor there
[] Add more features to the File Condition
[] Start adding UTs
[] Linter
[] Generator of templates
[] Move to client x server design a la Docker. Use unix sock
[x] Change a lot of the keys so we can have multiple services on same condition. End user should have no need to understand impl details.
[] Split file and dir into two separate services(?)
[] context vs. channels + start/stop methods. HTTP Server as a basis. Not sure that I need a context at all. We want all services to shutdown gracefully, not just cancel what is currently in progress.
[] context should be propagated to executors, they should be cancelled.
[] Interpret `~` as $HOME globally
