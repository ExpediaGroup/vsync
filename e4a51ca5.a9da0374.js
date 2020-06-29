(window.webpackJsonp=window.webpackJsonp||[]).push([[15],{154:function(e,t,n){"use strict";n.r(t),n.d(t,"frontMatter",(function(){return o})),n.d(t,"metadata",(function(){return c})),n.d(t,"rightToc",(function(){return l})),n.d(t,"default",(function(){return p}));var r=n(2),a=n(9),i=(n(0),n(158)),o={id:"origin",title:"Origin",sidebar_label:"Origin"},c={id:"internals/origin",isDocsHomePage:!1,title:"Origin",description:"This is the start of unidirectional connection for syncing secrets. It should point to primary vault cluster from which users expect the secrets to be propagated to other vaults in different regions.",source:"@site/docs/internals/origin.md",permalink:"/vsync/docs/internals/origin",editUrl:"https://github.com/ExpediaGroup/vsync/edit/master/website/docs/internals/origin.md",sidebar_label:"Origin",sidebar:"someSidebar",previous:{title:"Keywords",permalink:"/vsync/docs/internals/keywords"},next:{title:"Destination",permalink:"/vsync/docs/internals/destination"}},l=[{value:"Startup",id:"startup",children:[]},{value:"Cycle",id:"cycle",children:[]}],s={rightToc:l};function p(e){var t=e.components,n=Object(a.a)(e,["components"]);return Object(i.b)("wrapper",Object(r.a)({},s,n,{components:t,mdxType:"MDXLayout"}),Object(i.b)("p",null,"This is the start of unidirectional connection for syncing secrets. It should point to primary vault cluster from which users expect the secrets to be propagated to other vaults in different regions."),Object(i.b)("hr",null),Object(i.b)("h2",{id:"startup"},"Startup"),Object(i.b)("h4",{id:"step-1"},"Step 1"),Object(i.b)("p",null,"Get consul and vault clients pointing to origin"),Object(i.b)("h4",{id:"step-2"},"Step 2"),Object(i.b)("p",null,"Check if we could read, write, update, delete in origin consul kv under sync path"),Object(i.b)("h4",{id:"step-3"},"Step 3"),Object(i.b)("p",null,"Check if we could read, write, update, delete in origin vault under data paths specified in config"),Object(i.b)("h4",{id:"step-4"},"Step 4"),Object(i.b)("p",null,"Prepare an error channel through which anyone under sync cycle can contact to throw errors"),Object(i.b)("p",null,"We also need to listen to error channel and check if the error at hand is fatal or not."),Object(i.b)("p",null,"If not fatal, log the error with as much context available.\nIf fatal, stop the current sync cycle cleanly and future cycles. Log the error, inform a human, halt the program."),Object(i.b)("h4",{id:"step-5"},"Step 5"),Object(i.b)("p",null,"Prepare an signal channel through which OS can send halt signals. Useful for humans to stop the whole sync program cleanly stop."),Object(i.b)("h4",{id:"step-6"},"Step 6"),Object(i.b)("p",null,"A ticker is initialized for an interval (default: 1m) to start the sync cycle.\nThe trigger will be starting point for one cycle."),Object(i.b)("hr",null),Object(i.b)("h2",{id:"cycle"},"Cycle"),Object(i.b)("h4",{id:"step-0"},"Step 0"),Object(i.b)("p",null,"A timer with timeout (default: 5m) will be created for every sync cycle. If workers get struck inbetween or something happens we do not halt vsync. Instead we wait till the timeout and kill everything created for current sync cycle. "),Object(i.b)("h4",{id:"step-1-1"},"Step 1"),Object(i.b)("p",null,"Create a fresh ",Object(i.b)("inlineCode",{parentName:"p"},"sync info")," to store vsync metadata. It needs to be safe for concurrent usage."),Object(i.b)("h4",{id:"step-2-1"},"Step 2"),Object(i.b)("p",null,"For an interval (default: 1m) we get a list of paths recursively that needs to be synced based on data paths. Example, for datapath ",Object(i.b)("inlineCode",{parentName:"p"},"secret/")," we get absolute paths ",Object(i.b)("inlineCode",{parentName:"p"},"[secret/metadata/stage/app1, secret/metadata/stage/app2]")),Object(i.b)("h4",{id:"step-3-1"},"Step 3"),Object(i.b)("p",null,"We create multiple worker go routines (default: 1). Each worker will generate insight and save in sync info for a given absolute path."),Object(i.b)("p",null,"Each routine will be given:"),Object(i.b)("ul",null,Object(i.b)("li",{parentName:"ul"},"vault client pointing to origin"),Object(i.b)("li",{parentName:"ul"},"shared sync info"),Object(i.b)("li",{parentName:"ul"},"error channel"),Object(i.b)("li",{parentName:"ul"},"multiple absolute paths but one at a time")),Object(i.b)("p",null,"sync info needs be safe for concurrent usage"),Object(i.b)("h4",{id:"step-4-1"},"Step 4"),Object(i.b)("p",null,"Create 1 go routine to handle saving info to consul"),Object(i.b)("ul",null,Object(i.b)("li",{parentName:"ul"},"if cycle is successful, save consul sync info"),Object(i.b)("li",{parentName:"ul"},"if cycle has failed, abort saving info because it will corrupt existing sync info")),Object(i.b)("h4",{id:"step-5-1"},"Step 5"),Object(i.b)("p",null,"From the list of absolute paths send one path to next available worker. Once we have sent all the paths, wait for all worker go routines to complete their work."),Object(i.b)("p",null,"The sender needs to be in separate routine, because we need to stop sending work to worker if we get halt signals."),Object(i.b)("h4",{id:"step-6-1"},"Step 6"),Object(i.b)("p",null,"Reindex the sync info, for generating index info for each bucket."),Object(i.b)("h4",{id:"step-7"},"Step 7"),Object(i.b)("p",null,"If everything is successful, send save signal for saving info ( index and buckets ) to consul."),Object(i.b)("p",null,"If the cycle is aborted by signal, do not send the save signal for saving."),Object(i.b)("p",null,"We need to cleanly close the cycle. Log appropriate cycle messages."))}p.isMDXComponent=!0},158:function(e,t,n){"use strict";n.d(t,"a",(function(){return u})),n.d(t,"b",(function(){return h}));var r=n(0),a=n.n(r);function i(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function o(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function c(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?o(Object(n),!0).forEach((function(t){i(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):o(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function l(e,t){if(null==e)return{};var n,r,a=function(e,t){if(null==e)return{};var n,r,a={},i=Object.keys(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||(a[n]=e[n]);return a}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(a[n]=e[n])}return a}var s=a.a.createContext({}),p=function(e){var t=a.a.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):c(c({},t),e)),n},u=function(e){var t=p(e.components);return a.a.createElement(s.Provider,{value:t},e.children)},b={inlineCode:"code",wrapper:function(e){var t=e.children;return a.a.createElement(a.a.Fragment,{},t)}},d=a.a.forwardRef((function(e,t){var n=e.components,r=e.mdxType,i=e.originalType,o=e.parentName,s=l(e,["components","mdxType","originalType","parentName"]),u=p(n),d=r,h=u["".concat(o,".").concat(d)]||u[d]||b[d]||i;return n?a.a.createElement(h,c(c({ref:t},s),{},{components:n})):a.a.createElement(h,c({ref:t},s))}));function h(e,t){var n=arguments,r=t&&t.mdxType;if("string"==typeof e||r){var i=n.length,o=new Array(i);o[0]=d;var c={};for(var l in t)hasOwnProperty.call(t,l)&&(c[l]=t[l]);c.originalType=e,c.mdxType="string"==typeof e?e:r,o[1]=c;for(var s=2;s<i;s++)o[s]=n[s];return a.a.createElement.apply(null,o)}return a.a.createElement.apply(null,n)}d.displayName="MDXCreateElement"}}]);