# Reflection

For the current milestone 2 implementation, a single visitor with a very large body won't block the tunnel writer, because writes are chunked into small frames and the mutex is released between each one. But the problem occurs if a visitor is slow to recieve, the response bytes queue up in relayd's memory waiting for that visitor's TCP buffer to drain and my current design has no way to tell the sender to slow down. Under sustained load with a slow visitor, memory would grow. HTTP/2 solves this with flow control (WINDOW_UPDATE frames + per-stream credit windows), but I don't fully understand the mechanism yet, Parked for later.

