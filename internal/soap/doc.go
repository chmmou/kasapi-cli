// Package soap implements the codec for the KAS-API SOAP shape.
//
// Responses use the Apache xml-soap "ns2:Map" representation: every value
// carries an explicit xsi:type discriminator (xsd:string / xsd:int /
// xsd:float / xsd:boolean / ns2:Map / SOAP-ENC:Array). The package exposes
// a Value type that mirrors that shape and a Decode entry point for the
// SOAP envelope. SOAP-ENV:Fault bodies surface as *FaultError.
//
// Requests are a JSON payload wrapped in <tns:KasApi><Params>{json}</Params>.
// EncodeRequest produces a valid envelope from the typed Request struct.
//
// See issue #3 for the original design.
package soap
