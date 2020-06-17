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
 *   !proto.URLRequest,
 *   !proto.URLResponse>}
 */
const methodDescriptor_Shortener_GetURL = new grpc.web.MethodDescriptor(
  '/Shortener/GetURL',
  grpc.web.MethodType.UNARY,
  proto.URLRequest,
  proto.URLResponse,
  /**
   * @param {!proto.URLRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.URLResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.URLRequest,
 *   !proto.URLResponse>}
 */
const methodInfo_Shortener_GetURL = new grpc.web.AbstractClientBase.MethodInfo(
  proto.URLResponse,
  /**
   * @param {!proto.URLRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.URLResponse.deserializeBinary
);


/**
 * @param {!proto.URLRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.URLResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.URLResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.ShortenerClient.prototype.getURL =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/Shortener/GetURL',
      request,
      metadata || {},
      methodDescriptor_Shortener_GetURL,
      callback);
};


/**
 * @param {!proto.URLRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.URLResponse>}
 *     A native promise that resolves to the response
 */
proto.ShortenerPromiseClient.prototype.getURL =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/Shortener/GetURL',
      request,
      metadata || {},
      methodDescriptor_Shortener_GetURL);
};


module.exports = proto;

