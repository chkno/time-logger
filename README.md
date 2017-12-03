## A simple time-tracking tool

`$ go run tl.go`

Visit http://localhost:29804/log to record stuff, http://localhost:29804/view to see your stuff.

The killer feature of this tool: the backing store of the information collected and displayed by this tool is a simple text file. Inevitably, you will forget to log something or make a typo, and want to edit the log to correct it. With this tool, you can use the full power and comfort of your favorite $EDITOR on `tl.log`

![An example slice of the author's life as captured by this tool](https://raw.github.com/chkno/time-logger/master/example.png)
