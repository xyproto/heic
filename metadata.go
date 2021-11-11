package heic

/*
#cgo pkg-config: libheif
#include <libheif/heif.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"bytes"
	"encoding/base64"
	"errors"
	"unsafe"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"howett.net/plist"
)

type MetadataID uint

func (h *ImageHandle) MetadataCount() int {
	n := int(C.heif_image_handle_get_number_of_metadata_blocks(h.handle, nil))
	keepAlive(h)
	return n
}

func (h *ImageHandle) MetadataIDs() []MetadataID {
	nMeta := h.MetadataCount()
	if nMeta == 0 {
		return []MetadataID{}
	}
	meta := make([]C.uint, nMeta)

	C.heif_image_handle_get_list_of_metadata_block_IDs(h.handle, nil, &meta[0], C.int(nMeta))
	keepAlive(h)
	metaDataIDs := make([]MetadataID, nMeta)
	for i := 0; i < nMeta; i++ {
		metaDataIDs[i] = MetadataID(meta[i])
	}
	return metaDataIDs
}

func (h *ImageHandle) Metadata(mID MetadataID) []byte {
	nMeta := h.MetadataCount()
	if nMeta == 0 {
		return []byte{}
	}

	nData := C.heif_image_handle_get_metadata_size(h.handle, C.uint(mID))
	keepAlive(h)

	data := C.malloc(C.sizeof_char * nData)
	defer C.free(unsafe.Pointer(data))

	C.heif_image_handle_get_metadata(h.handle, C.uint(mID), data)
	keepAlive(h)

	return C.GoBytes(data, C.int(nData))
	//mExif.load(exifData+4, nData-4);
}

func (h *ImageHandle) ExifCount() int {
	filter := C.CString("Exif")
	defer C.free(unsafe.Pointer(filter))
	n := int(C.heif_image_handle_get_number_of_metadata_blocks(h.handle, filter))
	keepAlive(h)
	return n
}

func (h *ImageHandle) ExifIDs() []MetadataID {
	nMeta := h.ExifCount()
	if nMeta == 0 {
		return []MetadataID{}
	}
	filter := C.CString("Exif")
	defer C.free(unsafe.Pointer(filter))
	meta := make([]C.uint, nMeta)
	C.heif_image_handle_get_list_of_metadata_block_IDs(h.handle, filter, &meta[0], C.int(nMeta))
	keepAlive(h)
	metaDataIDs := make([]MetadataID, nMeta)
	for i := 0; i < nMeta; i++ {
		metaDataIDs[i] = MetadataID(meta[i])
	}
	return metaDataIDs
}

// MetadataMap takes a metadata ID and an XPath expression
// Within the metadata there may be XML, within the XML there may be a Base64 encoded string,
// and within the Base64 encoded string there may be a propery list in the Apple plist format
// and within that plist there may be a timestamp or solar position,
// encoded in the form of a property list. Oh joy.
func (h *ImageHandle) MetadataMap(mID MetadataID, xs string) (map[string]interface{}, error) {
	xmlData := bytes.ReplaceAll(h.Metadata(mID), []byte{0}, []byte{})
	expr, err := xpath.Compile(xs)
	if err != nil {
		return nil, err
	}
	doc, err := xmlquery.Parse(bytes.NewReader(xmlData))
	if err != nil {
		return nil, err
	}
	base64string, ok := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(string)
	if !ok {
		return nil, errors.New("could not find string at " + xs)
	}
	b64decoded, err := base64.StdEncoding.DecodeString(base64string)
	if err != nil {
		return nil, err
	}
	decodedMap := make(map[string]interface{})
	_, err = plist.Unmarshal(b64decoded, &decodedMap)
	if err != nil {
		return nil, err
	}
	return decodedMap, nil
}

func (h *ImageHandle) AppleTime(mID MetadataID) (map[string]interface{}, error) {
	return h.MetadataMap(mID, "string(//x:xmpmeta/rdf:RDF/rdf:Description/@apple_desktop:h24)")
}

func (h *ImageHandle) AppleSolar(mID MetadataID) (map[string]interface{}, error) {
	return h.MetadataMap(mID, "string(//x:xmpmeta/rdf:RDF/rdf:Description/@apple_desktop:solar)")
}
