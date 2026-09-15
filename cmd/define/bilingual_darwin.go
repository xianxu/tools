//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreServices -framework CoreFoundation
#include <CoreServices/CoreServices.h>
#include <dlfcn.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

typedef CFSetRef (*bi_available_fn)(void);
typedef CFStringRef (*bi_identifier_fn)(void *);
typedef CFArrayRef (*bi_search_fn)(void *, CFStringRef, CFIndex, CFIndex);
typedef CFStringRef (*bi_data_fn)(void *, CFIndex);
static bi_available_fn bi_available;
static bi_identifier_fn bi_identifier;
static bi_search_fn bi_search;
static bi_data_fn bi_data;
static pthread_once_t bi_once = PTHREAD_ONCE_INIT;
static void bi_resolve(void) {
 bi_available=(bi_available_fn)dlsym(RTLD_DEFAULT,"DCSCopyAvailableDictionaries");
 bi_identifier=(bi_identifier_fn)dlsym(RTLD_DEFAULT,"DCSDictionaryGetIdentifier");
 bi_search=(bi_search_fn)dlsym(RTLD_DEFAULT,"DCSCopyRecordsForSearchString");
 bi_data=(bi_data_fn)dlsym(RTLD_DEFAULT,"DCSRecordCopyData");
}

typedef struct { char *html; char *text; } bi_record;
typedef struct { int status; CFIndex count; bi_record *records; } bi_result;
// Status: 0 success, 1 absent dictionary, 2 absent API, 3 native failure,
// 4 bounded-result failure, 5 malformed native representation.
static char *bi_copy_string(CFStringRef s, CFIndex limit, int *status) {
 if (!s) { *status=5; return NULL; }
 CFIndex used=0, length=CFStringGetLength(s);
 CFIndex converted=CFStringGetBytes(s,CFRangeMake(0,length),kCFStringEncodingUTF8,0,false,NULL,0,&used);
 if (converted!=length) { *status=5; return NULL; }
 if (used>limit) { *status=4; return NULL; }
 char *out=malloc(used+1);
 if (!out) { *status=3; return NULL; }
 if (!CFStringGetCString(s,out,used+1,kCFStringEncodingUTF8)) {free(out);*status=5;return NULL;}
 return out;
}
static void bi_free(bi_result *r) {
 if (!r) return;
 if (r->records) for(CFIndex i=0;i<r->count;i++){free(r->records[i].html);free(r->records[i].text);}
 free(r->records);free(r);
}
static bi_result *bi_lookup(const char *identifier,const char *word,CFIndex record_limit,CFIndex byte_limit) {
 bi_result *out=calloc(1,sizeof(bi_result));
 if(!out)return NULL;
 pthread_once(&bi_once,bi_resolve);
 if(!bi_available||!bi_identifier||!bi_search||!bi_data){out->status=2;return out;}
 CFSetRef set=bi_available();
 if(!set){out->status=3;return out;}
 CFIndex count=CFSetGetCount(set);
 const void **values=malloc(sizeof(void *)*(count ? count : 1));
 if(!values){CFRelease(set);out->status=3;return out;}
 CFSetGetValues(set,values);
 CFStringRef wanted=CFStringCreateWithCString(NULL,identifier,kCFStringEncodingUTF8);
 void *dictionary=NULL;
 for(CFIndex i=0;i<count && wanted;i++) {
  CFStringRef id=bi_identifier((void *)values[i]);
  if(id && CFEqual(id,wanted)){dictionary=(void *)values[i];break;}
 }
 free(values);
 if(wanted)CFRelease(wanted);
 if(!dictionary){CFRelease(set);out->status=1;return out;}
 CFStringRef query=CFStringCreateWithCString(NULL,word,kCFStringEncodingUTF8);
 if(!query){CFRelease(set);out->status=3;return out;}
 // Request one extra record: a full cap cannot establish completeness.
 CFArrayRef records=bi_search(dictionary,query,0,record_limit+1);
 CFRelease(query);
 if(!records){CFRelease(set);return out;}
 out->count=CFArrayGetCount(records);
 if(out->count>record_limit){out->status=4;out->count=0;CFRelease(records);CFRelease(set);return out;}
 out->records=calloc(out->count ? out->count : 1,sizeof(bi_record));
 if(!out->records){out->status=3;out->count=0;CFRelease(records);CFRelease(set);return out;}
 for(CFIndex i=0;i<out->count;i++) {
  void *record=(void *)CFArrayGetValueAtIndex(records,i);
  CFStringRef html=bi_data(record,0);
  out->records[i].html=bi_copy_string(html,byte_limit,&out->status);
  if(html)CFRelease(html);
  if(out->status)break;
  CFStringRef text=bi_data(record,3);
  out->records[i].text=bi_copy_string(text,byte_limit,&out->status);
  if(text)CFRelease(text);
  if(out->status)break;
 }
 CFRelease(records);CFRelease(set);
 return out;
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// Limits are configurable only at this internal seam for live boundary checks;
// the constructor always uses the production bounds. No CF handle is retained
// across calls, so installation changes are observed on the next lookup.
type nativeSpanishEnglishSource struct {
	dictionaryID           string
	recordLimit, byteLimit int
}

func newSpanishEnglishSource() recordSource {
	return nativeSpanishEnglishSource{spanishEnglishDictionaryID, bilingualMaxRecords, bilingualMaxBytes}
}

func (s nativeSpanishEnglishSource) Records(word string) ([]bilingualRecord, error) {
	if len(word) > bilingualMaxBytes {
		return nil, ErrBilingualLimit
	}
	if !utf8.ValidString(word) || strings.ContainsRune(word, 0) {
		return nil, ErrBilingualMalformed
	}
	id := C.CString(s.dictionaryID)
	defer C.free(unsafe.Pointer(id))
	query := C.CString(word)
	defer C.free(unsafe.Pointer(query))
	result := C.bi_lookup(id, query, C.CFIndex(s.recordLimit), C.CFIndex(s.byteLimit))
	if result == nil {
		return nil, ErrLookupFailed
	}
	defer C.bi_free(result)
	var err error
	switch result.status {
	case 1:
		err = ErrBilingualUnavailable
	case 2:
		err = ErrBilingualUnsupported
	case 3:
		err = ErrLookupFailed
	case 4:
		err = ErrBilingualLimit
	case 5:
		err = ErrBilingualMalformed
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.dictionaryID, err)
	}
	out := make([]bilingualRecord, int(result.count))
	for i, record := range unsafe.Slice(result.records, int(result.count)) {
		out[i] = bilingualRecord{HTML: C.GoString(record.html), Text: C.GoString(record.text)}
	}
	return out, nil
}
