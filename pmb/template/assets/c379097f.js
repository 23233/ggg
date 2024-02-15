import{g as R,a as N,a9 as D,s as y,U as l,_ as i,aa as M,r as U,u as z,b as I,j as g,d as E,P as e,a3 as F,e as K}from"./988f4188.js";function W(r){return R("MuiCircularProgress",r)}N("MuiCircularProgress",["root","determinate","indeterminate","colorPrimary","colorSecondary","svg","circle","circleDeterminate","circleIndeterminate","circleDisableShrink"]);const B=["className","color","disableShrink","size","style","thickness","value","variant"];let d=r=>r,P,S,$,w;const t=44,G=D(P||(P=d`
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
`)),L=D(S||(S=d`
  0% {
    stroke-dasharray: 1px, 200px;
    stroke-dashoffset: 0;
  }

  50% {
    stroke-dasharray: 100px, 200px;
    stroke-dashoffset: -15px;
  }

  100% {
    stroke-dasharray: 100px, 200px;
    stroke-dashoffset: -125px;
  }
`)),V=r=>{const{classes:s,variant:a,color:o,disableShrink:u}=r,h={root:["root",a,`color${l(o)}`],svg:["svg"],circle:["circle",`circle${l(a)}`,u&&"circleDisableShrink"]};return K(h,W,s)},Y=y("span",{name:"MuiCircularProgress",slot:"Root",overridesResolver:(r,s)=>{const{ownerState:a}=r;return[s.root,s[a.variant],s[`color${l(a.color)}`]]}})(({ownerState:r,theme:s})=>i({display:"inline-block"},r.variant==="determinate"&&{transition:s.transitions.create("transform")},r.color!=="inherit"&&{color:(s.vars||s).palette[r.color].main}),({ownerState:r})=>r.variant==="indeterminate"&&M($||($=d`
      animation: ${0} 1.4s linear infinite;
    `),G)),Z=y("svg",{name:"MuiCircularProgress",slot:"Svg",overridesResolver:(r,s)=>s.svg})({display:"block"}),q=y("circle",{name:"MuiCircularProgress",slot:"Circle",overridesResolver:(r,s)=>{const{ownerState:a}=r;return[s.circle,s[`circle${l(a.variant)}`],a.disableShrink&&s.circleDisableShrink]}})(({ownerState:r,theme:s})=>i({stroke:"currentColor"},r.variant==="determinate"&&{transition:s.transitions.create("stroke-dashoffset")},r.variant==="indeterminate"&&{strokeDasharray:"80px, 200px",strokeDashoffset:0}),({ownerState:r})=>r.variant==="indeterminate"&&!r.disableShrink&&M(w||(w=d`
      animation: ${0} 1.4s ease-in-out infinite;
    `),L)),T=U.forwardRef(function(s,a){const o=z({props:s,name:"MuiCircularProgress"}),{className:u,color:h="primary",disableShrink:_=!1,size:f=40,style:j,thickness:n=3.6,value:m=0,variant:k="indeterminate"}=o,O=I(o,B),c=i({},o,{color:h,disableShrink:_,size:f,thickness:n,value:m,variant:k}),p=V(c),v={},b={},x={};if(k==="determinate"){const C=2*Math.PI*((t-n)/2);v.strokeDasharray=C.toFixed(3),x["aria-valuenow"]=Math.round(m),v.strokeDashoffset=`${((100-m)/100*C).toFixed(3)}px`,b.transform="rotate(-90deg)"}return g(Y,i({className:E(p.root,u),style:i({width:f,height:f},b,j),ownerState:c,ref:a,role:"progressbar"},x,O,{children:g(Z,{className:p.svg,ownerState:c,viewBox:`${t/2} ${t/2} ${t} ${t}`,children:g(q,{className:p.circle,style:v,ownerState:c,cx:t,cy:t,r:(t-n)/2,fill:"none",strokeWidth:n})})}))});T.propTypes={classes:e.object,className:e.string,color:e.oneOfType([e.oneOf(["inherit","primary","secondary","error","info","success","warning"]),e.string]),disableShrink:F(e.bool,r=>r.disableShrink&&r.variant&&r.variant!=="indeterminate"?new Error("MUI: You have provided the `disableShrink` prop with a variant other than `indeterminate`. This will have no effect."):null),size:e.oneOfType([e.number,e.string]),style:e.object,sx:e.oneOfType([e.arrayOf(e.oneOfType([e.func,e.object,e.bool])),e.func,e.object]),thickness:e.number,value:e.number,variant:e.oneOf(["determinate","indeterminate"])};const H=T;export{H as C};
