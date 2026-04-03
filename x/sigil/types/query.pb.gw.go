// Hand-written gRPC-gateway for x/sigil query service.
// Maps REST paths to gRPC QueryClient calls.
package types

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/utilities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

var (
	pattern_Query_Sigil_0       = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 1, 1, 0, 4, 1, 5, 3}, []string{"oasyce", "sigil", "v1", "sigil_id"}, "", runtime.AssumeColonVerbOpt(true)))
	pattern_Query_Bond_0        = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3, 1, 0, 4, 1, 5, 4}, []string{"oasyce", "sigil", "v1", "bond", "bond_id"}, "", runtime.AssumeColonVerbOpt(true)))
	pattern_Query_BondsBySigil  = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3, 1, 0, 4, 1, 5, 4}, []string{"oasyce", "sigil", "v1", "bonds_by_sigil", "sigil_id"}, "", runtime.AssumeColonVerbOpt(true)))
	pattern_Query_Lineage       = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3, 1, 0, 4, 1, 5, 4}, []string{"oasyce", "sigil", "v1", "lineage", "sigil_id"}, "", runtime.AssumeColonVerbOpt(true)))
	pattern_Query_ActiveCount   = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3}, []string{"oasyce", "sigil", "v1", "active_count"}, "", runtime.AssumeColonVerbOpt(true)))
	pattern_Query_Params_0      = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3}, []string{"oasyce", "sigil", "v1", "params"}, "", runtime.AssumeColonVerbOpt(true)))

	forward_Query = runtime.ForwardResponseMessage
)

// RegisterQueryHandlerClient registers the http handlers for service Query to "mux".
func RegisterQueryHandlerClient(ctx context.Context, mux *runtime.ServeMux, client QueryClient) error {
	mux.Handle("GET", pattern_Query_Sigil_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		var protoReq QuerySigilRequest
		val, ok := pathParams["sigil_id"]
		if !ok {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "missing parameter %s", "sigil_id"))
			return
		}
		protoReq.SigilId, err = runtime.String(val)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "sigil_id", err))
			return
		}
		resp, md, err := request_Sigil(rctx, inboundMarshaler, client, &protoReq)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle("GET", pattern_Query_Bond_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		var protoReq QueryBondRequest
		val, ok := pathParams["bond_id"]
		if !ok {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "missing parameter %s", "bond_id"))
			return
		}
		protoReq.BondId, err = runtime.String(val)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "bond_id", err))
			return
		}
		resp, md, err := request_Bond(rctx, inboundMarshaler, client, &protoReq)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle("GET", pattern_Query_BondsBySigil, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		var protoReq QueryBondsBySigilRequest
		val, ok := pathParams["sigil_id"]
		if !ok {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "missing parameter %s", "sigil_id"))
			return
		}
		protoReq.SigilId, err = runtime.String(val)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "sigil_id", err))
			return
		}
		resp, md, err := request_BondsBySigil(rctx, inboundMarshaler, client, &protoReq)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle("GET", pattern_Query_Lineage, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		var protoReq QueryLineageRequest
		val, ok := pathParams["sigil_id"]
		if !ok {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "missing parameter %s", "sigil_id"))
			return
		}
		protoReq.SigilId, err = runtime.String(val)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "sigil_id", err))
			return
		}
		resp, md, err := request_Lineage(rctx, inboundMarshaler, client, &protoReq)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle("GET", pattern_Query_ActiveCount, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_ActiveCount(rctx, inboundMarshaler, client, &QueryActiveCountRequest{})
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle("GET", pattern_Query_Params_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_Params(rctx, inboundMarshaler, client, &QueryParamsRequest{})
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_Query(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	return nil
}

// request helpers — call the gRPC client and return (response, metadata, error).

func request_Sigil(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QuerySigilRequest) (*QuerySigilResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.Sigil(ctx, req)
	return resp, metadata, err
}

func request_Bond(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QueryBondRequest) (*QueryBondResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.Bond(ctx, req)
	return resp, metadata, err
}

func request_BondsBySigil(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QueryBondsBySigilRequest) (*QueryBondsBySigilResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.BondsBySigil(ctx, req)
	return resp, metadata, err
}

func request_Lineage(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QueryLineageRequest) (*QueryLineageResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.Lineage(ctx, req)
	return resp, metadata, err
}

func request_ActiveCount(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QueryActiveCountRequest) (*QueryActiveCountResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.ActiveCount(ctx, req)
	return resp, metadata, err
}

func request_Params(ctx context.Context, _ runtime.Marshaler, client QueryClient, req *QueryParamsRequest) (*QueryParamsResponse, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata
	resp, err := client.Params(ctx, req)
	return resp, metadata, err
}

// Suppress unused import errors.
var _ = utilities.NewDoubleArray
var _ = grpclog.Infof
