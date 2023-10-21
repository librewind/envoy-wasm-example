package main

import (
    "encoding/json"
    "github.com/percivalalb/sipuri"

    "github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
    "github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

const (
    sipUriHeader = "X-Sip-Uri"
    kamailioHeader = "X-Kamailio"
    parameterName = "kamailio"
)

func main() {
    proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
    // Embed the default VM context here,
    // so that we don't need to reimplement all the methods.
    types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
    return &pluginContext{}
}

type pluginContext struct {
    // Embed the default plugin context here,
    // so that we don't need to reimplement all the methods.
    types.DefaultPluginContext
}

// Override types.DefaultPluginContext.
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
    return &httpHeaders{
        contextID: contextID,
    }
}

func (p *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
    proxywasm.LogDebug("loading plugin config")

    return types.OnPluginStartStatusOK
}

type httpHeaders struct {
    // Embed the default http context here,
    // so that we don't need to reimplement all the methods.
    types.DefaultHttpContext
    contextID             uint32
    totalRequestBodySize  int
    totalResponseBodySize int
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
    path, err := proxywasm.GetHttpRequestHeader(":path")
    if err != nil {
        proxywasm.LogInfof("Header not found in request: :path")
        return types.ActionContinue
    }
    proxywasm.LogInfof("path: %s", path)

    header, err := proxywasm.GetHttpRequestHeader(sipUriHeader)
    if err != nil {
        proxywasm.LogInfof("Header not found in request: %s", sipUriHeader)
        return types.ActionContinue
    }
    proxywasm.LogInfof("header: %s", header)
    parameterValue := getParameterFromSipUri(header, parameterName)

    proxywasm.RemoveHttpRequestHeader(sipUriHeader)

    if parameterValue != "" {
        err2 := proxywasm.ReplaceHttpRequestHeader(kamailioHeader, parameterValue)
        if err2 != nil {
            proxywasm.LogCriticalf("failed to set request header: %s", kamailioHeader)
        }
    }

    return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
    ctx.totalRequestBodySize += bodySize
    if !endOfStream {
        // Wait until we see the entire body to replace.
        return types.ActionPause
    }

    originalBody, err := proxywasm.GetHttpRequestBody(0, ctx.totalRequestBodySize)
    if err != nil {
        proxywasm.LogErrorf("failed to get request body: %v", err)
        return types.ActionContinue
    }
    
    sipUri := getSipUriFromBody(originalBody)
    if sipUri == "" {
        proxywasm.LogErrorf("failed to get SIP URI")
        return types.ActionContinue
    }

    proxywasm.LogInfof("SIP URI: %s", sipUri)
    kamailio := getParameterFromSipUri(sipUri, parameterName)
    proxywasm.LogInfof("kamailio: %s", kamailio)

    // Очень жаль что из функции OnHttpRequestBody нет доступа к HTTP заголовкам
    // Так бы можно было установить HTTP заголовок на основе содержимого в body

    return types.ActionContinue
}

func getSipUriFromBody(body []byte) string {
    var content map[string]any
    if err := json.Unmarshal(body, &content); err != nil {
        proxywasm.LogErrorf("failed to unmarshal json: %v", err)
        return ""
    }

    if _, exists := content["uri"]; exists == false {
        proxywasm.LogError("content does not contian SIP URI")
        return ""
    }

    return content["uri"].(string)
}

func getParameterFromSipUri(sipUri string, parameter string) string {
    sipURI, err := sipuri.Parse(sipUri)
    if err != nil {
        proxywasm.LogErrorf("failed to parse SIP URI: %v", err)
        return ""
    }

    return sipURI.Params().Get(parameter)
}

