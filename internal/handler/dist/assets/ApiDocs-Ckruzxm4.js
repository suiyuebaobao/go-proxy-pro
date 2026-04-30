import{r as k,o as K,j as L,b as I,d as C,e as n,f as l,w as o,i as u,p as s,t,g as f,ae as R,I as z,L as q,A as D,af as B,D as M,V as G,ag as U,ah as j,ai as N}from"./index-BgPRS0ga.js";import{_ as E}from"./_plugin-vue_export-helper-DlAUqK2U.js";const V={class:"api-docs"},X={class:"section-title"},Y={class:"code-inline"},F={class:"section-title"},Q={class:"platform-grid"},J={class:"platform-name"},W={class:"platform-models"},Z={class:"section-title"},$={class:"code-inline endpoint-code"},ee={class:"section-title"},ne={class:"api-detail"},le={class:"endpoint-banner"},oe={class:"code-block"},se={class:"api-detail"},te={class:"endpoint-banner"},ae={class:"code-block"},ie={class:"api-detail"},de={class:"endpoint-banner"},pe={class:"code-inline"},re={class:"code-block"},me={class:"code-block"},ue={class:"api-detail"},ce={class:"endpoint-banner"},fe={class:"code-block"},ge={class:"api-detail"},ye={class:"endpoint-banner"},ve={class:"code-block"},ke={class:"code-block"},xe={class:"api-detail"},_e={class:"code-block"},be={class:"code-block"},Ae={class:"code-block"},Ie={class:"section-title"},Ce={class:"section-title"},Pe={class:"code-inline"},we={class:"section-title"},Te={__name:"ApiDocs",setup(Oe){const A=k("claude"),d=k(window.location.origin),x=k(""),p=k("your-api-key");K(async()=>{var _;try{const e=await L.getApiKeys(),r=((_=e==null?void 0:e.data)==null?void 0:_.items)||(e==null?void 0:e.data)||[];Array.isArray(r)&&r.length>0&&(x.value=r[0].key||"",x.value&&(p.value=x.value.substring(0,8)+"..."))}catch{}});const P=[{name:"Claude (Anthropic)",models:"claude-sonnet-4, claude-opus-4 等",type:"claude"},{name:"OpenAI",models:"gpt-4o, gpt-4.1, o3 等",type:"openai"},{name:"Google Gemini",models:"gemini-2.5-pro, gemini-2.5-flash 等",type:"gemini"},{name:"DeepSeek",models:"deepseek-chat, deepseek-reasoner 等",type:"compat"},{name:"通义千问 (Qwen)",models:"qwen-turbo, qwen-max 等",type:"compat"},{name:"智谱 GLM",models:"glm-4, glm-4-flash 等",type:"compat"},{name:"Kimi (月之暗面)",models:"moonshot-v1-8k 等",type:"compat"},{name:"豆包 (字节)",models:"doubao-pro, doubao-lite 等",type:"compat"},{name:"零一万物",models:"yi-large, yi-medium 等",type:"compat"},{name:"阶跃星辰",models:"step-1-8k, step-2-16k 等",type:"compat"},{name:"讯飞星火",models:"spark-lite, spark-pro 等",type:"compat"},{name:"xAI (Grok)",models:"grok-2, grok-3 等",type:"compat"},{name:"Mistral",models:"mistral-large 等",type:"compat"},{name:"Cohere",models:"command-r-plus 等",type:"compat"},{name:"SiliconFlow",models:"硅基流动托管模型",type:"compat"},{name:"自定义 API",models:"任意 OpenAI 兼容接口",type:"custom"}],w=[{platform:"Claude",endpoint:"/claude/v1/messages",format:"Claude Messages API",tagType:""},{platform:"OpenAI + 所有兼容平台",endpoint:"/openai/v1/chat/completions",format:"OpenAI Chat Completions",tagType:"success"},{platform:"Gemini",endpoint:"/gemini/v1/chat",format:"Gemini Chat",tagType:"warning"},{platform:"Codex CLI / Claude Code",endpoint:"/v1/responses",format:"OpenAI Responses API",tagType:"info"}],T=[{prefix:"deepseek-*",platform:"DeepSeek",example:"deepseek-chat, deepseek-reasoner"},{prefix:"qwen-*",platform:"通义千问 (Qwen)",example:"qwen-turbo, qwen-plus, qwen-max"},{prefix:"glm-*",platform:"智谱 GLM",example:"glm-4, glm-4-flash"},{prefix:"moonshot-*",platform:"Kimi (月之暗面)",example:"moonshot-v1-8k"},{prefix:"doubao-*",platform:"豆包 (字节)",example:"doubao-pro-4k"},{prefix:"yi-*",platform:"零一万物",example:"yi-large, yi-medium"},{prefix:"Baichuan*",platform:"百川",example:"Baichuan2-Turbo"},{prefix:"minimax-*",platform:"MiniMax",example:"minimax-abab6.5"},{prefix:"step-*",platform:"阶跃星辰",example:"step-1-8k"},{prefix:"spark-*",platform:"讯飞星火",example:"spark-v3.5"},{prefix:"grok-*",platform:"xAI Grok",example:"grok-2"},{prefix:"mistral-*",platform:"Mistral",example:"mistral-large-latest"},{prefix:"command-*",platform:"Cohere",example:"command-r-plus"}],O=[{code:401,desc:"API Key 无效/缺失/过期",action:"检查 API Key 是否正确，是否已过期"},{code:402,desc:"配额/次数耗尽",action:"联系管理员充值、升级套餐或增加请求次数"},{code:403,desc:"权限不足 / IP 被禁止",action:"检查 API Key 权限和 IP 白名单设置"},{code:429,desc:"请求速率超限",action:"降低请求频率，参考响应头 Retry-After"},{code:502,desc:"上游 API 错误",action:"稍后重试，或尝试更换模型"},{code:503,desc:"无可用账户",action:"联系管理员检查后端账户池"}],h=[{name:"x-api-key",required:!0,desc:"API Key 认证（与 Authorization 二选一）"},{name:"Authorization",required:!0,desc:"Bearer Token 认证，格式: Bearer your-key"},{name:"Content-Type",required:!0,desc:"必须为 application/json"},{name:"x-session-id",required:!1,desc:"会话绑定，同一 session 路由到同一后端账户（Claude Code 自动发送）"}];return(_,e)=>{const r=u("el-icon"),b=u("el-step"),H=u("el-steps"),y=u("el-alert"),c=u("el-card"),m=u("el-tag"),i=u("el-table-column"),v=u("el-table"),g=u("el-tab-pane"),S=u("el-tabs");return C(),I("div",V,[e[58]||(e[58]=n("div",{class:"page-header"},[n("h2",null,"API 接入文档"),n("p",{class:"subtitle"},"了解如何将 AI 能力集成到你的应用中")],-1)),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",X,[l(r,null,{default:o(()=>[l(f(R))]),_:1}),e[1]||(e[1]=n("span",null,"快速开始",-1))])]),default:o(()=>[l(H,{active:3,"align-center":"",class:"quick-steps"},{default:o(()=>[l(b,{title:"创建 API Key",description:"在「我的 API Key」页面创建"}),l(b,{title:"选择平台端点",description:"根据目标模型选择对应端点"}),l(b,{title:"发送请求",description:"携带 API Key 调用接口"})]),_:1}),l(y,{type:"info",closable:!1,"show-icon":"",class:"base-url-alert"},{title:o(()=>[n("span",null,[e[2]||(e[2]=s("基础 URL：",-1)),n("code",Y,t(d.value),1)])]),default:o(()=>[...e[3]||(e[3]=[n("span",null,[s("所有接口路径均相对于此 URL。认证 Header："),n("code",{class:"code-inline"},"x-api-key: your-key"),s(" 或 "),n("code",{class:"code-inline"},"Authorization: Bearer your-key")],-1)])]),_:1})]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",F,[l(r,null,{default:o(()=>[l(f(B))]),_:1}),e[4]||(e[4]=n("span",null,"支持的 AI 平台",-1))])]),default:o(()=>[e[5]||(e[5]=n("p",{style:{color:"var(--el-text-color-secondary)",margin:"0 0 16px 0"}},"本平台支持以下 AI 服务商，通过统一的 API Key 即可调用所有平台的模型：",-1)),n("div",Q,[(C(),I(z,null,q(P,a=>n("div",{key:a.name,class:D(["platform-chip",a.type])},[n("span",J,t(a.name),1),n("span",W,t(a.models),1)],2)),64))])]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",Z,[l(r,null,{default:o(()=>[l(f(M))]),_:1}),e[6]||(e[6]=n("span",null,"接口端点",-1))])]),default:o(()=>[l(y,{type:"warning",closable:!1,"show-icon":"",style:{"margin-bottom":"16px"}},{title:o(()=>[...e[7]||(e[7]=[s("所有 OpenAI 兼容平台（DeepSeek、通义千问、GLM、Kimi 等）均通过 ",-1),n("code",{class:"code-inline"},"/openai/v1/chat/completions",-1),s(" 端点访问，系统根据模型名自动路由到对应平台。",-1)])]),_:1}),l(v,{data:w,stripe:"",style:{width:"100%"}},{default:o(()=>[l(i,{prop:"platform",label:"适用平台",width:"200"},{default:o(({row:a})=>[l(m,{type:a.tagType,size:"small"},{default:o(()=>[s(t(a.platform),1)]),_:2},1032,["type"])]),_:1}),l(i,{prop:"method",label:"方法",width:"80"},{default:o(()=>[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[8]||(e[8]=[s("POST",-1)])]),_:1})]),_:1}),l(i,{prop:"endpoint",label:"端点"},{default:o(({row:a})=>[n("code",$,t(a.endpoint),1)]),_:1}),l(i,{prop:"format",label:"请求格式",width:"220"})]),_:1})]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",ee,[l(r,null,{default:o(()=>[l(f(G))]),_:1}),e[9]||(e[9]=n("span",null,"接口详情与示例",-1))])]),default:o(()=>[l(S,{modelValue:A.value,"onUpdate:modelValue":e[0]||(e[0]=a=>A.value=a),type:"border-card"},{default:o(()=>[l(g,{label:"Claude",name:"claude"},{default:o(()=>[n("div",ne,[n("div",le,[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[10]||(e[10]=[s("POST",-1)])]),_:1}),e[11]||(e[11]=n("code",null,"/claude/v1/messages",-1))]),e[16]||(e[16]=n("p",null,[s("与 "),n("a",{href:"https://docs.anthropic.com/en/api/messages",target:"_blank"},"Claude Messages API"),s(" 完全兼容。")],-1)),e[17]||(e[17]=n("h4",null,"请求体",-1)),e[18]||(e[18]=n("pre",{class:"code-block"},[n("code",null,`{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "你好，请介绍一下自己"}
  ],
  "stream": false
}`)],-1)),e[19]||(e[19]=n("h4",null,"cURL 示例",-1)),n("pre",oe,[n("code",null,"curl -X POST "+t(d.value)+`/claude/v1/messages \\
  -H "x-api-key: `+t(p.value)+`" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'`,1)]),e[20]||(e[20]=n("h4",null,"响应示例",-1)),e[21]||(e[21]=n("pre",{class:"code-block"},[n("code",null,`{
  "id": "msg_xxxx",
  "type": "message",
  "role": "assistant",
  "model": "claude-sonnet-4-20250514",
  "content": [{"type": "text", "text": "你好！我是 Claude..."}],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 12,
    "output_tokens": 45
  }
}`)],-1)),l(y,{type:"info",closable:!1,"show-icon":"",style:{"margin-top":"12px"}},{title:o(()=>[...e[12]||(e[12]=[s("会话粘性",-1)])]),default:o(()=>[e[13]||(e[13]=s(" 添加 ",-1)),e[14]||(e[14]=n("code",{class:"code-inline"},"x-session-id: your-session-id",-1)),e[15]||(e[15]=s(" 请求头可保证同一会话始终路由到同一后端账户。 ",-1))]),_:1})])]),_:1}),l(g,{label:"OpenAI",name:"openai"},{default:o(()=>[n("div",se,[n("div",te,[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[22]||(e[22]=[s("POST",-1)])]),_:1}),e[23]||(e[23]=n("code",null,"/openai/v1/chat/completions",-1))]),e[24]||(e[24]=n("p",null,[s("与 "),n("a",{href:"https://platform.openai.com/docs/api-reference/chat",target:"_blank"},"OpenAI Chat Completions API"),s(" 完全兼容。")],-1)),e[25]||(e[25]=n("h4",null,"请求体",-1)),e[26]||(e[26]=n("pre",{class:"code-block"},[n("code",null,`{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}`)],-1)),e[27]||(e[27]=n("h4",null,"cURL 示例",-1)),n("pre",ae,[n("code",null,"curl -X POST "+t(d.value)+`/openai/v1/chat/completions \\
  -H "Authorization: Bearer `+t(p.value)+`" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'`,1)]),e[28]||(e[28]=n("h4",null,"响应示例",-1)),e[29]||(e[29]=n("pre",{class:"code-block"},[n("code",null,`{
  "id": "chatcmpl-xxxx",
  "object": "chat.completion",
  "model": "gpt-4o",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "Hello! How can I help you?"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 20, "completion_tokens": 9, "total_tokens": 29}
}`)],-1))])]),_:1}),l(g,{label:"多平台",name:"multiplatform"},{default:o(()=>[n("div",ie,[n("div",de,[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[30]||(e[30]=[s("POST",-1)])]),_:1}),e[32]||(e[32]=n("code",null,"/openai/v1/chat/completions",-1)),l(m,{type:"warning",size:"small",style:{"margin-left":"8px"}},{default:o(()=>[...e[31]||(e[31]=[s("同一端点",-1)])]),_:1})]),l(y,{type:"info",closable:!1,"show-icon":"",style:{"margin-bottom":"12px"}},{title:o(()=>[...e[33]||(e[33]=[s("与 OpenAI 共用同一端点",-1)])]),default:o(()=>[e[34]||(e[34]=s(" DeepSeek、通义千问、GLM、Kimi、豆包等平台均兼容 OpenAI 格式，通过同一个端点访问，系统根据模型名自动路由到对应平台。 ",-1))]),_:1}),e[35]||(e[35]=n("h4",null,"模型路由表",-1)),e[36]||(e[36]=n("p",null,"直接在 model 字段填写对应平台的模型名即可，系统自动识别：",-1)),l(v,{data:T,stripe:"",size:"small",style:{margin:"12px 0"}},{default:o(()=>[l(i,{prop:"prefix",label:"模型前缀 / 名称",width:"200"},{default:o(({row:a})=>[n("code",pe,t(a.prefix),1)]),_:1}),l(i,{prop:"platform",label:"路由到平台"}),l(i,{prop:"example",label:"示例模型"})]),_:1}),e[37]||(e[37]=n("p",null,[s("也可用 "),n("code",{class:"code-inline"},"平台,模型名"),s(" 格式显式指定，例如 "),n("code",{class:"code-inline"},'"model": "deepseek,deepseek-chat"')],-1)),e[38]||(e[38]=n("h4",null,"DeepSeek 示例",-1)),n("pre",re,[n("code",null,"curl -X POST "+t(d.value)+`/openai/v1/chat/completions \\
  -H "Authorization: Bearer `+t(p.value)+`" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "你好"}]
  }'`,1)]),e[39]||(e[39]=n("h4",null,"通义千问示例",-1)),n("pre",me,[n("code",null,"curl -X POST "+t(d.value)+`/openai/v1/chat/completions \\
  -H "Authorization: Bearer `+t(p.value)+`" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "qwen-turbo",
    "messages": [{"role": "user", "content": "你好"}]
  }'`,1)])])]),_:1}),l(g,{label:"Gemini",name:"gemini"},{default:o(()=>[n("div",ue,[n("div",ce,[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[40]||(e[40]=[s("POST",-1)])]),_:1}),e[41]||(e[41]=n("code",null,"/gemini/v1/chat",-1))]),e[42]||(e[42]=n("h4",null,"请求体",-1)),e[43]||(e[43]=n("pre",{class:"code-block"},[n("code",null,`{
  "model": "gemini-2.5-pro",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}`)],-1)),e[44]||(e[44]=n("h4",null,"cURL 示例",-1)),n("pre",fe,[n("code",null,"curl -X POST "+t(d.value)+`/gemini/v1/chat \\
  -H "x-api-key: `+t(p.value)+`" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-2.5-pro",
    "messages": [{"role": "user", "content": "Hello"}]
  }'`,1)])])]),_:1}),l(g,{label:"Codex CLI",name:"codex"},{default:o(()=>[n("div",ge,[n("div",ye,[l(m,{type:"success",size:"small",effect:"dark"},{default:o(()=>[...e[45]||(e[45]=[s("POST",-1)])]),_:1}),e[46]||(e[46]=n("code",null,"/v1/responses",-1))]),e[47]||(e[47]=n("p",null,[s("兼容 OpenAI Responses API，主要用于 "),n("strong",null,"Codex CLI"),s(" 和 "),n("strong",null,"Claude Code"),s(" 等编程工具。")],-1)),e[48]||(e[48]=n("p",null,[s("以下端点均支持："),n("code",{class:"code-inline"},"/responses"),s("、"),n("code",{class:"code-inline"},"/v1/responses"),s("、"),n("code",{class:"code-inline"},"/openai/v1/responses")],-1)),e[49]||(e[49]=n("h4",null,"Codex CLI 配置",-1)),n("pre",ve,[n("code",null,`# 设置环境变量
export OPENAI_API_KEY="`+t(p.value)+`"
export OPENAI_BASE_URL="`+t(d.value)+`"

# 使用 Codex CLI
codex "explain this code"`,1)]),e[50]||(e[50]=n("h4",null,"Claude Code 配置",-1)),n("pre",ke,[n("code",null,`# 设置环境变量
export ANTHROPIC_API_KEY="`+t(p.value)+`"
export ANTHROPIC_BASE_URL="`+t(d.value)+`"

# Claude Code 会自动使用 /claude/v1/messages 端点`,1)])])]),_:1}),l(g,{label:"SDK 示例",name:"sdk"},{default:o(()=>[n("div",xe,[e[51]||(e[51]=n("h4",null,"Python - OpenAI SDK",-1)),n("pre",_e,[n("code",null,`from openai import OpenAI

client = OpenAI(
    api_key="`+t(p.value)+`",
    base_url="`+t(d.value)+`/openai/v1"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello"}]
)

print(response.choices[0].message.content)`,1)]),e[52]||(e[52]=n("h4",null,"Python - Anthropic SDK",-1)),n("pre",be,[n("code",null,`import anthropic

client = anthropic.Anthropic(
    api_key="`+t(p.value)+`",
    base_url="`+t(d.value)+`/claude"
)

message = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)

print(message.content[0].text)`,1)]),e[53]||(e[53]=n("h4",null,"Node.js - OpenAI SDK",-1)),n("pre",Ae,[n("code",null,`import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: '`+t(p.value)+`',
  baseURL: '`+t(d.value)+`/openai/v1',
});

const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello' }],
});

console.log(response.choices[0].message.content);`,1)])])]),_:1})]),_:1},8,["modelValue"])]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",Ie,[l(r,null,{default:o(()=>[l(f(U))]),_:1}),e[54]||(e[54]=n("span",null,"错误码参考",-1))])]),default:o(()=>[l(v,{data:O,stripe:""},{default:o(()=>[l(i,{prop:"code",label:"HTTP 状态码",width:"140"},{default:o(({row:a})=>[l(m,{type:a.code>=500?"danger":a.code>=400?"warning":"info",size:"small"},{default:o(()=>[s(t(a.code),1)]),_:2},1032,["type"])]),_:1}),l(i,{prop:"desc",label:"说明",width:"250"}),l(i,{prop:"action",label:"建议处理"})]),_:1})]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",Ce,[l(r,null,{default:o(()=>[l(f(j))]),_:1}),e[55]||(e[55]=n("span",null,"请求头参考",-1))])]),default:o(()=>[l(v,{data:h,stripe:""},{default:o(()=>[l(i,{prop:"name",label:"Header",width:"220"},{default:o(({row:a})=>[n("code",Pe,t(a.name),1)]),_:1}),l(i,{prop:"required",label:"必选",width:"80"},{default:o(({row:a})=>[l(m,{type:a.required?"danger":"info",size:"small"},{default:o(()=>[s(t(a.required?"是":"否"),1)]),_:2},1032,["type"])]),_:1}),l(i,{prop:"desc",label:"说明"})]),_:1})]),_:1}),l(c,{shadow:"never",class:"doc-section"},{header:o(()=>[n("div",we,[l(r,null,{default:o(()=>[l(f(N))]),_:1}),e[56]||(e[56]=n("span",null,"注意事项",-1))])]),default:o(()=>[e[57]||(e[57]=n("ul",{class:"notes-list"},[n("li",null,[n("strong",null,"模型可用性"),s("：可用的模型取决于管理员配置的后端账户池，并非所有模型名称都保证可用")]),n("li",null,[n("strong",null,"Token 用量"),s("：返回的 Token 数量可能经过倍率调整，反映的是计费 Token")]),n("li",null,[n("strong",null,"请求超时"),s("：流式请求默认无超时，非流式请求建议设置 60-120 秒超时")]),n("li",null,[n("strong",null,"智能重试"),s("：系统内置自动重试（默认 3 次），一个账户失败会自动切换到其他账户")]),n("li",null,[n("strong",null,"并发限制"),s("：每个用户和 API Key 可能配置了并发上限，超出返回 429")]),n("li",null,[n("strong",null,"速率限制"),s("：配置了 RPM/RPD 后，响应头会包含 "),n("code",{class:"code-inline"},"X-RateLimit-Limit"),s(" 和 "),n("code",{class:"code-inline"},"X-RateLimit-Remaining")])],-1))]),_:1})])}}},Se=E(Te,[["__scopeId","data-v-133c6466"]]);export{Se as default};
