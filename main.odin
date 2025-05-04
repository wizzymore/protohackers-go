package main

import "core:fmt"
import "core:sync/chan"
import "core:thread"
import "core:time"

main :: proc() {
	c, _ := chan.create(chan.Chan(i32), context.temp_allocator)

	t := thread.create_and_start_with_poly_data(c, run_thread, context)
	defer thread.destroy(t)

	chan.send(c, 10)
	chan.send(c, 20)
	chan.close(c)
	thread.join(t)
}

run_thread :: proc(c: chan.Chan(i32)) {
	defer fmt.println("thread closed")
	for {
		i, ok := chan.recv(c)
		if !ok {
			return
		}
		fmt.println(i)
	}
}
