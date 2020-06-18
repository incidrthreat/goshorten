/**
 * @fileoverview gRPC-Web generated client stub for 
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!



const grpc = {};
grpc.web = require('grpc-web');

const proto = require('./url_service_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.ShortenerClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.ShortenerPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.ShortURLReq,
 *   !proto.ShortURLResp>}
 */
const methodDescriptor_Shortener_CreateURL = new grpc.web.MethodDescriptor(
  '/Shortener/CreateURL',
  grpc.web.MethodType.UNARY,
  proto.ShortURLReq,
  proto.ShortURLResp,
  /**
   * @param {!proto.ShortURLReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.ShortURLResp.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.ShortURLReq,
 *   !proto.ShortURLResp>}
 */
const methodInfo_Shortener_CreateURL = new grpc.web.AbstractClientBase.MethodInfo(
  proto.ShortURLResp,
  /**
   * @param {!proto.ShortURLReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.ShortURLResp.deserializeBinary
);


/**
 * @param {!proto.ShortURLReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.ShortURLResp)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.ShortURLResp>|undefined}
 *     The XHR Node Readable Stream
 */
proto.ShortenerClient.prototype.createURL =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/Shortener/CreateURL',
      request,
      metadata || {},
      methodDescriptor_Shortener_CreateURL,
      callback);
};


/**
 * @param {!proto.ShortURLReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.ShortURLResp>}
 *     A native promise that resolves to the response
 */
proto.ShortenerPromiseClient.prototype.createURL =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/Shortener/CreateURL',
      request,
      metadata || {},
      methodDescriptor_Shortener_CreateURL);
};


module.exports = proto;

