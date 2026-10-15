import {chromium} from '@playwright/test';
import {spawn} from 'node:child_process';
import {writeFileSync,createWriteStream,mkdirSync} from 'node:fs';
const root=process.env.LEARNING_PROOF_LOGS || '/workspace/cloud-setup/retro-screens/final-fix';
mkdirSync(root,{recursive:true});
const server=spawn('node',['.output/server/index.mjs'],{cwd:process.cwd(),env:{...process.env,PORT:'3103',HOST:'127.0.0.1',NUXT_PUBLIC_API_BASE:'http://127.0.0.1:3199'},stdio:['ignore','pipe','pipe']});
const log=createWriteStream(root+'/focused-server.log');server.stdout.pipe(log);server.stderr.pipe(log);
let browser;
try {
for(let i=0;i<60;i++){try{if((await fetch('http://127.0.0.1:3103')).ok)break}catch{ /* Production server is starting. */ }await new Promise(r=>setTimeout(r,250))}
browser=await chromium.launch({executablePath:'/usr/bin/chromium'});
const context=await browser.newContext({viewport:{width:1280,height:900},reducedMotion:'reduce'});
await context.addInitScript(()=>localStorage.setItem('aelp.auth',JSON.stringify({accessToken:'fixture',expiresAt:Date.now()+86400000,user:{id:'owner',full_name:'Review',email:'r@example.invalid',cefr_current:'B1'}})));
const page=await context.newPage();
const outline={roadmap_id:'r',title:'Review roadmap',cefr_level:'B1',today:'2026-10-18',day_number:18,modules:[1,2,3,4].map(week=>({week,title:'Region '+week,focus:'Real focus',days:Array.from({length:7},(_,i)=>{let n=(week-1)*7+i+1;return {day_number:n,date:'2026-10-'+String(n).padStart(2,'0'),title:'Day '+n,minutes_spent:0,is_target_met:false,tasks:[]}})}))};
const prefs={notification_time:'07:30:00',timezone:'UTC',next_reminder_at:'2026-10-15T07:30:00Z',push_configured:false};
let releaseSettings;let deferSettings=false;
await page.route('**/api/v1/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path==='/api/v1/settings'&&deferSettings)await new Promise(r=>releaseSettings=r);
 const body=path==='/api/v1/roadmap'?outline:path==='/api/v1/settings'?prefs:path==='/api/v1/settings/notifications'?{status:'updated',notification_time:'09:15:00',next_reminder_at:'2026-10-15T09:15:00Z'}:path==='/api/v1/pet/status'?{plant_name:'Mầm',health_points:80,stage:'sprout',current_streak:1,last_practiced_at:null}:{date:'2026-10-14',day_number:14,total_minutes_required:30,accumulated_seconds:0,is_target_met:false,tasks:[]};
 await route.fulfill({json:body});
});
await page.goto('http://127.0.0.1:3103/roadmap');await page.getByRole('heading',{name:'Review roadmap',exact:true}).waitFor();
const metrics=await page.locator('main').evaluate(el=>({width:el.getBoundingClientRect().width,left:el.getBoundingClientRect().left,paddingLeft:getComputedStyle(el).paddingLeft,maxWidth:getComputedStyle(el).maxWidth}));
await page.screenshot({path:root+'/focused-roadmap-desktop.png',fullPage:true});
await page.goto('http://127.0.0.1:3103/settings');await page.getByLabel('Giờ nhắc',{exact:true}).waitFor();
await page.getByRole('link',{name:'‹ Trại'}).click();await page.waitForURL('**/');
deferSettings=true;
await page.getByRole('link',{name:/Cài đặt tài khoản/}).click();
for(let i=0;i<50&&!releaseSettings;i++)await new Promise(r=>setTimeout(r,20));
await page.getByLabel('Giờ nhắc',{exact:true}).fill('09:15');await page.getByRole('button',{name:'Lưu giờ nhắc'}).click();
await page.getByText(/Giờ đã lưu: 09:15/).waitFor();
const afterSave=await page.getByLabel('Giờ nhắc',{exact:true}).inputValue();const staleResponse=page.waitForResponse(res=>new URL(res.url()).pathname==='/api/v1/settings');releaseSettings();
await staleResponse;
await page.waitForTimeout(100);
if(await page.getByLabel('Giờ nhắc',{exact:true}).inputValue()!=='09:15'||!await page.getByText(/Giờ đã lưu: 09:15/).isVisible())throw Error('stale GET replaced confirmed save');
const afterStaleGet=await page.getByLabel('Giờ nhắc',{exact:true}).inputValue();
await page.getByRole('link',{name:'‹ Trại'}).click();await page.waitForURL('**/');
releaseSettings=undefined;
await page.getByRole('link',{name:/Cài đặt tài khoản/}).click();
for(let i=0;i<50&&!releaseSettings;i++)await new Promise(r=>setTimeout(r,20));
await page.getByLabel('Giờ nhắc',{exact:true}).fill('10:45');
const dirtyResponse=page.waitForResponse(res=>new URL(res.url()).pathname==='/api/v1/settings');releaseSettings();await dirtyResponse;await page.waitForTimeout(100);
if(await page.getByLabel('Giờ nhắc',{exact:true}).inputValue()!=='10:45'||!await page.getByText(/Giờ đã lưu: 07:30/).isVisible())throw Error('entry refresh discarded dirty edit');
// Route navigation preserves the accepted memory store; no direct store mutation.
await page.getByRole('link',{name:'‹ Trại'}).click();await page.waitForURL('**/');
await page.getByRole('link',{name:/hành trình/}).first().click();
await page.getByRole('region',{name:'Nhiệm vụ ngày 18'}).waitFor();
await page.waitForTimeout(150);
const entry=await page.locator('#region-day-18').evaluate(el=>({top:el.getBoundingClientRect().top,bottom:el.getBoundingClientRect().bottom,scroll:scrollY}));
if(entry.scroll<=0||entry.top>=900||entry.bottom<=0)throw Error('cached online entry did not position today '+JSON.stringify(entry));
await page.getByRole('link',{name:'‹ Trại'}).click();await page.waitForURL('**/');
await context.setOffline(true);
await page.getByRole('link',{name:/hành trình/}).first().click();
await page.getByText('Đang ngoại tuyến — bản đồ là bản đã lưu.').waitFor();await page.getByRole('region',{name:'Nhiệm vụ ngày 18'}).waitFor();
await page.waitForTimeout(100);
const offlineEntry=await page.locator('#region-day-18').evaluate(el=>({top:el.getBoundingClientRect().top,bottom:el.getBoundingClientRect().bottom,scroll:scrollY}));
if(offlineEntry.scroll<=0||offlineEntry.top>=900||offlineEntry.bottom<=0)throw Error('cached offline entry did not position today');
await context.setOffline(false);
const geometries=[];
for(const width of [375,1280]){
 await page.setViewportSize({width,height:900});
 const g=await page.locator('main').evaluate(el=>{const r=el.getBoundingClientRect(),c=getComputedStyle(el),p=el.querySelector('[id^="region-day-"]').getBoundingClientRect();return {width:r.width,left:r.left,paddingLeft:parseFloat(c.paddingLeft),paddingRight:parseFloat(c.paddingRight),regionLeft:p.left,regionRight:p.right}});
 if(Math.abs(g.width-Math.min(width,448))>1||Math.abs(g.left-(width-g.width)/2)>1||g.paddingLeft!==16||g.paddingRight!==16||Math.abs(g.regionLeft-g.left-16)>1||Math.abs(g.regionRight-g.left-g.width+16)>1)throw Error('wrong roadmap geometry '+JSON.stringify(g));
 geometries.push({viewport:width,...g});
}
writeFileSync(root+'/focused-results.json',JSON.stringify({roadmap:metrics,geometries,entry,offlineEntry,settings:{afterSave,afterStaleGet,dirtyEdit:'10:45'}},null,2));
console.log(JSON.stringify({roadmap:metrics,geometries,entry,offlineEntry,settings:{afterSave,afterStaleGet,dirtyEdit:'10:45'}}));
}finally{if(browser)await browser.close();server.kill('SIGTERM');log.end()}
