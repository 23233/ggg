import{j as n,O as p,p as b,r as g}from"./a765d8e4.js";import{u as M,f as h,M as x}from"./e1cfc97f.js";import{A as j,a as v}from"./1ed0e1db.js";const _=({children:l})=>{const{updateSettings:d,settings:f}=b(),{data:e,error:m,isLoading:c}=M("get_models",async()=>{const r=await j.get("models");return await v(r,"获取所有模型失败")},{refreshInterval:60*1e3,focusThrottleInterval:10*1e3});return g.useEffect(()=>{var r;if(e){const t=h(f.menus.items,{title:"模型管理"});t&&(t.subMenus=[],(r=e==null?void 0:e.models)==null||r.map(s=>{var o,u,a,i;(i=t.subMenus)==null||i.push({title:s.info.alias||((o=s.info)==null?void 0:o.path_id)||((u=s.info)==null?void 0:u.table_name)||((a=s.info)==null?void 0:a.unique_id),group:s.info.group,search:{name:s.info.unique_id},link:"/models/task"})}),d({menus:{items:{subMenus:t.subMenus}}}))}},[e]),n(x.Provider,{value:{models:(e==null?void 0:e.models)||[],isLoading:c,error:m},children:l})},P=()=>n(_,{children:n(p,{})});export{P as default};