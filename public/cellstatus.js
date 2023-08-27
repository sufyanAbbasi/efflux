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

goog.provide('proto.efflux.CellStatus');

goog.require('jspb.BinaryReader');
goog.require('jspb.BinaryWriter');
goog.require('jspb.Message');

goog.forwardDeclare('proto.efflux.CellActionStatus');
goog.forwardDeclare('proto.efflux.CellType');
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
proto.efflux.CellStatus = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.efflux.CellStatus.repeatedFields_, null);
};
goog.inherits(proto.efflux.CellStatus, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.efflux.CellStatus.displayName = 'proto.efflux.CellStatus';
}

/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.efflux.CellStatus.repeatedFields_ = [8,9,10,11,12];



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
proto.efflux.CellStatus.prototype.toObject = function(opt_includeInstance) {
  return proto.efflux.CellStatus.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.efflux.CellStatus} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.efflux.CellStatus.toObject = function(includeInstance, msg) {
  var f, obj = {
    timestamp: jspb.Message.getFieldWithDefault(msg, 1, 0),
    cellType: jspb.Message.getFieldWithDefault(msg, 2, 0),
    name: jspb.Message.getFieldWithDefault(msg, 3, ""),
    renderId: jspb.Message.getFieldWithDefault(msg, 4, ""),
    damage: jspb.Message.getFieldWithDefault(msg, 5, 0),
    spawnTime: jspb.Message.getFieldWithDefault(msg, 6, 0),
    viralLoad: jspb.Message.getFieldWithDefault(msg, 7, 0),
    transportPathList: (f = jspb.Message.getRepeatedField(msg, 8)) == null ? undefined : f,
    wantPathList: (f = jspb.Message.getRepeatedField(msg, 9)) == null ? undefined : f,
    proteinsList: (f = jspb.Message.getRepeatedField(msg, 10)) == null ? undefined : f,
    presentedList: (f = jspb.Message.getRepeatedField(msg, 11)) == null ? undefined : f,
    cellActionsList: (f = jspb.Message.getRepeatedField(msg, 12)) == null ? undefined : f
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
 * @return {!proto.efflux.CellStatus}
 */
proto.efflux.CellStatus.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.efflux.CellStatus;
  return proto.efflux.CellStatus.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.efflux.CellStatus} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.efflux.CellStatus}
 */
proto.efflux.CellStatus.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTimestamp(value);
      break;
    case 2:
      var value = /** @type {!proto.efflux.CellType} */ (reader.readEnum());
      msg.setCellType(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setRenderId(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setDamage(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setSpawnTime(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setViralLoad(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.addTransportPath(value);
      break;
    case 9:
      var value = /** @type {string} */ (reader.readString());
      msg.addWantPath(value);
      break;
    case 10:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedUint32() : [reader.readUint32()]);
      for (var i = 0; i < values.length; i++) {
        msg.addProteins(values[i]);
      }
      break;
    case 11:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedUint32() : [reader.readUint32()]);
      for (var i = 0; i < values.length; i++) {
        msg.addPresented(values[i]);
      }
      break;
    case 12:
      var values = /** @type {!Array<!proto.efflux.CellActionStatus>} */ (reader.isDelimited() ? reader.readPackedEnum() : [reader.readEnum()]);
      for (var i = 0; i < values.length; i++) {
        msg.addCellActions(values[i]);
      }
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
proto.efflux.CellStatus.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.efflux.CellStatus.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.efflux.CellStatus} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.efflux.CellStatus.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTimestamp();
  if (f !== 0) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getCellType();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getRenderId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getDamage();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getSpawnTime();
  if (f !== 0) {
    writer.writeInt64(
      6,
      f
    );
  }
  f = message.getViralLoad();
  if (f !== 0) {
    writer.writeInt64(
      7,
      f
    );
  }
  f = message.getTransportPathList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      8,
      f
    );
  }
  f = message.getWantPathList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      9,
      f
    );
  }
  f = message.getProteinsList();
  if (f.length > 0) {
    writer.writePackedUint32(
      10,
      f
    );
  }
  f = message.getPresentedList();
  if (f.length > 0) {
    writer.writePackedUint32(
      11,
      f
    );
  }
  f = message.getCellActionsList();
  if (f.length > 0) {
    writer.writePackedEnum(
      12,
      f
    );
  }
};


/**
 * optional int64 timestamp = 1;
 * @return {number}
 */
proto.efflux.CellStatus.prototype.getTimestamp = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setTimestamp = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional CellType cell_type = 2;
 * @return {!proto.efflux.CellType}
 */
proto.efflux.CellStatus.prototype.getCellType = function() {
  return /** @type {!proto.efflux.CellType} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.efflux.CellType} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setCellType = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};


/**
 * optional string name = 3;
 * @return {string}
 */
proto.efflux.CellStatus.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string render_id = 4;
 * @return {string}
 */
proto.efflux.CellStatus.prototype.getRenderId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setRenderId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional int32 damage = 5;
 * @return {number}
 */
proto.efflux.CellStatus.prototype.getDamage = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setDamage = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional int64 spawn_time = 6;
 * @return {number}
 */
proto.efflux.CellStatus.prototype.getSpawnTime = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setSpawnTime = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};


/**
 * optional int64 viral_load = 7;
 * @return {number}
 */
proto.efflux.CellStatus.prototype.getViralLoad = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setViralLoad = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};


/**
 * repeated string transport_path = 8;
 * @return {!Array<string>}
 */
proto.efflux.CellStatus.prototype.getTransportPathList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 8));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setTransportPathList = function(value) {
  return jspb.Message.setField(this, 8, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.addTransportPath = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 8, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.clearTransportPathList = function() {
  return this.setTransportPathList([]);
};


/**
 * repeated string want_path = 9;
 * @return {!Array<string>}
 */
proto.efflux.CellStatus.prototype.getWantPathList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 9));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setWantPathList = function(value) {
  return jspb.Message.setField(this, 9, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.addWantPath = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 9, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.clearWantPathList = function() {
  return this.setWantPathList([]);
};


/**
 * repeated uint32 proteins = 10;
 * @return {!Array<number>}
 */
proto.efflux.CellStatus.prototype.getProteinsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 10));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setProteinsList = function(value) {
  return jspb.Message.setField(this, 10, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.addProteins = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 10, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.clearProteinsList = function() {
  return this.setProteinsList([]);
};


/**
 * repeated uint32 presented = 11;
 * @return {!Array<number>}
 */
proto.efflux.CellStatus.prototype.getPresentedList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 11));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setPresentedList = function(value) {
  return jspb.Message.setField(this, 11, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.addPresented = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 11, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.clearPresentedList = function() {
  return this.setPresentedList([]);
};


/**
 * repeated CellActionStatus cell_actions = 12;
 * @return {!Array<!proto.efflux.CellActionStatus>}
 */
proto.efflux.CellStatus.prototype.getCellActionsList = function() {
  return /** @type {!Array<!proto.efflux.CellActionStatus>} */ (jspb.Message.getRepeatedField(this, 12));
};


/**
 * @param {!Array<!proto.efflux.CellActionStatus>} value
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.setCellActionsList = function(value) {
  return jspb.Message.setField(this, 12, value || []);
};


/**
 * @param {!proto.efflux.CellActionStatus} value
 * @param {number=} opt_index
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.addCellActions = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 12, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.efflux.CellStatus} returns this
 */
proto.efflux.CellStatus.prototype.clearCellActionsList = function() {
  return this.setCellActionsList([]);
};

