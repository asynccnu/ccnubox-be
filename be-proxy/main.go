package main

func main() {
	server := InitGRPCServer()
	err := server.Serve()
	if err != nil {
		panic(err)
	}
}
