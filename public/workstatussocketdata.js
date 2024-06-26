// source: efflux.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

goog.provide('proto.efflux.WorkStatusSocketData');

goog.require('jspb.BinaryReader');
goog.require('jspb.BinaryWriter');
goog.require('jspb.Message');

/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.efflux.WorkStatusSocketData = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.efflux.WorkStatusSocketData, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.efflux.WorkStatusSocketData.displayName = 'proto.efflux.WorkStatusSocketData';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.efflux.WorkStatusSocketData.prototype.toObject = function(opt_includeInstance) {
  return proto.efflux.WorkStatusSocketData.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.efflux.WorkStatusSocketData} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.efflux.WorkStatusSocketData.toObject = function(includeInstance, msg) {
  var f, obj = {
    workType: jspb.Message.getFieldWithDefault(msg, 1, ""),
    requestCount: jspb.Message.getFieldWithDefault(msg, 2, 0),
    successCount: jspb.Message.getFieldWithDefault(msg, 3, 0),
    failureCount: jspb.Message.getFieldWithDefault(msg, 4, 0),
    completedCount: jspb.Message.getFieldWithDefault(msg, 5, 0),
    completedFailureCount: jspb.Message.getFieldWithDefault(msg, 6, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.efflux.WorkStatusSocketData}
 */
proto.efflux.WorkStatusSocketData.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.efflux.WorkStatusSocketData;
  return proto.efflux.WorkStatusSocketData.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.efflux.WorkStatusSocketData} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.efflux.WorkStatusSocketData}
 */
proto.efflux.WorkStatusSocketData.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkType(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRequestCount(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setSuccessCount(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFailureCount(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setCompletedCount(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setCompletedFailureCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.efflux.WorkStatusSocketData.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.efflux.WorkStatusSocketData.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.efflux.WorkStatusSocketData} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.efflux.WorkStatusSocketData.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWorkType();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRequestCount();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getSuccessCount();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getFailureCount();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getCompletedCount();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getCompletedFailureCount();
  if (f !== 0) {
    writer.writeInt32(
      6,
      f
    );
  }
};


/**
 * optional string work_type = 1;
 * @return {string}
 */
proto.efflux.WorkStatusSocketData.prototype.getWorkType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setWorkType = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int32 request_count = 2;
 * @return {number}
 */
proto.efflux.WorkStatusSocketData.prototype.getRequestCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setRequestCount = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional int32 success_count = 3;
 * @return {number}
 */
proto.efflux.WorkStatusSocketData.prototype.getSuccessCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setSuccessCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 failure_count = 4;
 * @return {number}
 */
proto.efflux.WorkStatusSocketData.prototype.getFailureCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setFailureCount = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional int32 completed_count = 5;
 * @return {number}
 */
proto.efflux.WorkStatusSocketData.prototype.getCompletedCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setCompletedCount = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional int32 completed_failure_count = 6;
 * @return {number}
 */
proto.efflux.WorkStatusSocketData.prototype.getCompletedFailureCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.WorkStatusSocketData} returns this
 */
proto.efflux.WorkStatusSocketData.prototype.setCompletedFailureCount = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};


