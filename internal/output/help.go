package output

import "fmt"

const (
	reset = "\x1b[0m"
	bold  = "\x1b[1m"
	dim   = "\x1b[2m"
	green = "\x1b[32m"
	cyan  = "\x1b[1;36m"
)

func PrintHelp() {
	fmt.Printf("%sUsage:%s mpulse [OPTIONS] <COMMAND>\n\n", bold, reset)
	fmt.Printf("%sCommands:%s\n", bold, reset)
	fmt.Printf("  %sversion%s      Show mpulse version\n", green, reset)                                   //绿色
	fmt.Printf("  %shelp%s         Show mpulse help text\n", green, reset)                                 //绿色
	fmt.Printf("  %smode%s         mode=cli: command line mode, mode=server: daemon mode\n", green, reset) //绿色
	fmt.Printf("  %shelp%s         Show mpulse help text\n\n", green, reset)                               //绿色

	fmt.Printf("%sOptions:%s\n", cyan, reset)                                                                          //青蓝
	fmt.Printf("  %s--config%s     %sAvoid reading from the cache%s\n", cyan, dim, reset, reset)                       //
	fmt.Printf("  %s--listen%s     %shttp listen port%s\n", cyan, dim, reset, reset)                                   //
	fmt.Printf("  %s--config%s     %sdesignated configuration file path%s\n", cyan, dim, reset, reset)                 //
	fmt.Printf("  %s--ports%s      %sList of ports to be monitored, separated by commas%s\n", cyan, dim, reset, reset) //
}
