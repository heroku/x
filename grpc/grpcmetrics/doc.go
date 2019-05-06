// Package grpcmetrics provides interceptors for collecting metrics about grpc servers and clients.
//
// Metrics are prefixed with:
//
//		grpc.{server,client}.<service>.<method>
//
// For example:
//
//		grpc.server.domain-streamer.put-domain
//		grpc.client.domain-streamer.stream-updates
//
// For each endpoint, clients and servers will report:
//
//		requests - counter of requests
//		request-duration.ms - histogram of request durations in milliseconds
//		errors - counter of requests which result in errors
//		response-codes.<code> - counter of grpc response codes
//
// In addition to these metrics, streams will also report:
//
//		stream.clients - gauge of connected clients
//		stream.sends - counter of sent messages
//		stream.sends.errors - counter of message send errors
//		stream.send-duration.ms - histogram of send durations in milliseconds
//		stream.recvs - counter of received messages
//		stream.recvs.errors - counter of message recv errors
//		stream.recv-duration.ms - histogram of recv durations in milliseconds
//
package grpcmetrics
