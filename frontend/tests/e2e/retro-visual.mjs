import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import { mkdirSync, createWriteStream, writeFileSync } from 'node:fs';
const root=process.env.LEARNING_PROOF_LOGS || '/workspace/cloud-setup/retro-screens/task7';
const shots=`${root}/screens`;
mkdirSync(shots,{recursive:true});
const log=createWriteStream(`${root}/visual-server.log`);
const server=spawn('node',['.output/server/index.mjs'],{cwd:process.cwd(),env:{...process.env,NUXT_PUBLIC_API_BASE:'http://127.0.0.1:3199',NUXT_PUBLIC_GOOGLE_CLIENT_ID:'fixture',PORT:'3102',HOST:'127.0.0.1'},detached:true,stdio:['ignore','pipe','pipe']});
server.stdout.pipe(log);server.stderr.pipe(log);
const origin='http://127.0.0.1:3102';
const daily={date:'2026-10-13',day_number:13,total_minutes_required:30,accumulated_seconds:600,is_target_met:false,tasks:['vocabulary','reading','practice'].map((task_type,i)=>({id:`task-${i}`,task_type,title:['Những từ mới cho cuộc họp hàng tuần','Đọc hiểu thông báo thay đổi lịch họp','Thực hành phản hồi lịch làm việc'][i],duration_minutes:10,is_completed:i===0,content_json:{}}))};
const pet={plant_name:'Mầm Non',health_points:80,stage:'sprout',current_streak:5,last_practiced_at:'2026-10-12T03:00:00Z'};
const map={roadmap_id:'map-fixture',title:'English for work and daily life',cefr_level:'B1',today:'2026-10-13',day_number:13,modules:[1,2,3,4].map(week=>({week,title:['Everyday conversations','Clear and polite work emails','Meetings and decisions','Putting everything together'][week-1],focus:'Practice useful English with real situations and clear examples.',days:Array.from({length:7},(_,j)=>{const n=(week-1)*7+j+1;return {day_number:n,date:`2026-10-${String(n).padStart(2,'0')}`,title:`Day ${n}: Useful English for work`,minutes_spent:n===1?30:n===2?12:n===13?10:0,is_target_met:n===1,tasks:['vocabulary','reading','practice'].map((task_type,i)=>({id:`day-${n}-task-${i}`,task_type,title:['Words for work','Read a short message','Choose a clear response'][i],duration_minutes:10,is_completed:n===1?true:n===2?null:false}))}})}))};
const settings={notification_time:'20:00:00',timezone:'Asia/Ho_Chi_Minh',next_reminder_at:'2026-10-14T13:00:00Z',push_configured:false};
const quiz={questions:Array.from({length:10},(_,i)=>({id:`q${i+1}`,prompt:i===0?'She ___ a teacher.':`Question ${i+1}`,options:{A:'am',B:'is',C:'are',D:'be'}}))};
let browser;
try {
 let ready=false;for(let i=0;i<90;i++){try{if((await fetch(origin)).ok){ready=true;break}}catch{ /* Server is still starting. */ }await new Promise(r=>setTimeout(r,500))}if(!ready)throw Error('visual server timeout');
 browser=await chromium.launch({executablePath:'/usr/bin/chromium',headless:true});
 const manifest=[];
 async function makePage(width,mode='normal',route='/'){
  const context=await browser.newContext({viewport:{width,height:width===1280?900:812},reducedMotion:'reduce'});
  const page=await context.newPage();
  if(!mode.startsWith('login')) await page.addInitScript(()=>localStorage.setItem('aelp.auth',JSON.stringify({accessToken:'fixture-token',expiresAt:Date.now()+86400000,user:{id:'fixture-owner',full_name:'Nguyễn Hoàng Anh Minh',email:'fixture@example.invalid',cefr_current:'B1'}})));
  const routePath=route;
  if(!mode.startsWith('login')) await page.route('**/api/v1/**',async route=>{const path=new URL(route.request().url()).pathname;const target={'/':'/quests/daily','/onboarding':'/onboarding/quiz','/roadmap':'/roadmap','/revive':'/pet/status','/settings':'/settings'}[routePath];if(mode==='offline'&&routePath==='/onboarding'&&path.endsWith(target))return route.abort('internetdisconnected');if(mode==='loading'&&path.endsWith(target))return new Promise(()=>{});if(mode==='error'&&path.endsWith(target))return route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'unavailable'})});if(mode==='google-error'&&path.endsWith('/integrations/google/sync'))return route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'google_unavailable'})});const body=path.endsWith('/pet/status')?(routePath==='/revive'?{...pet,health_points:0,stage:'wilted'}:pet):path.endsWith('/roadmap')?map:path.endsWith('/settings')?settings:path.endsWith('/onboarding/quiz')?quiz:daily;const absent=routePath==='/onboarding'&&path.endsWith('/quests/daily');return route.fulfill({status:absent?404:200,contentType:'application/json',body:JSON.stringify(absent?{error:'no_active_roadmap'}:body)})});
  return {page,context};
 }
 async function shot(route,width,mode='normal'){
  console.log(`Checking ${route} ${width} ${mode}`);
  const {page,context}=await makePage(width,mode,route);
  try {await page.goto(origin+route+(mode==='login-error'?'?code=fixture&state=wrong':''),{waitUntil:'domcontentloaded'});await page.waitForTimeout(700);
   const heading={'/login':/Học 30 phút/,'/onboarding':/Chọn hành trình/,'/':/Phòng hôm nay/,'/roadmap':/English for work and daily life/,'/revive':/Hồi sinh/,'/settings':/Cài đặt/}[route];
   if(heading&&!['loading','error'].includes(mode))await page.getByRole('heading',{name:heading}).first().waitFor({timeout:7000});
   if(route==='/onboarding'&&mode==='quiz'){await page.getByRole('radio',{name:'IELTS 7.0'}).click();await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).click();await page.getByText('Lượt 1 / 10').waitFor()}
   if(route==='/onboarding'&&['loading','error'].includes(mode)){await page.getByRole('radio',{name:'IELTS 7.0'}).click();await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).click()}
   if(route==='/settings'&&mode.startsWith('confirm')){await page.getByRole('button',{name:'Rời quán · Đăng xuất'}).click();await page.getByRole('dialog').waitFor()}
   if(route==='/settings'&&mode==='google-error'){await page.getByRole('button',{name:'Đồng bộ với Google'}).click();await page.getByText('Google chưa phản hồi. Thử lại sau ít phút.').waitFor()}
   if(mode.endsWith('keyboard')){
    await page.keyboard.press('Tab');
    if(!await page.evaluate(()=>document.activeElement?.matches(':focus-visible')))throw Error('keyboard focus is not visibly marked');
   }
   if(mode.includes('zoom200'))await page.evaluate(()=>{document.documentElement.style.zoom='200%'});
   if(mode==='loading'){
    if(route==='/onboarding'){await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).waitFor();if(!await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).isDisabled())throw Error('onboarding quiz loading must disable start')}
    else {await page.getByLabel('Đang tải').first().waitFor();if(route==='/'&&(await page.locator('[data-loading-tile]').count()!==3||await page.locator('[data-next-task]').count()))throw Error('hub loading must show three inert tiles without navigation')}
   }
   if(mode==='error'){
    const message={'/':'Không tải được nhiệm vụ. Thử lại.','/onboarding':'Không tải được bài kiểm tra. Thử lại.','/roadmap':'Không tải được hành trình. Thử lại.','/revive':'Không tải được trạng thái cây. Chưa thể biết cây có héo hay không.','/settings':'Chưa đọc được giờ nhắc đã lưu.'}[route];
    await page.getByText(message).waitFor();
    if(route==='/onboarding')await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).waitFor();
    else await page.getByRole('button',{name:'Thử lại'}).first().waitFor();
   }
   if(mode==='login-error')await page.getByRole('alert').getByText('Phiên đăng nhập không hợp lệ. Thử lại.').waitFor();
   if(mode==='offline'){
    await context.setOffline(true);await page.evaluate(()=>window.dispatchEvent(new Event('offline')));
    if(route==='/roadmap')await page.getByText('Đang ngoại tuyến — bản đồ là bản đã lưu.').waitFor();
    if(route==='/revive'){await page.getByText('Cần mạng để hồi sinh. Tiến độ đã lưu vẫn còn.').waitFor();if(!await page.getByRole('button',{name:'Hồi sinh (15 phút)'}).isDisabled())throw Error('offline revive action must be disabled')}
    if(route==='/settings'){await page.getByText('Cần mạng để lưu cài đặt.').waitFor();if(!await page.getByRole('button',{name:'Lưu giờ nhắc'}).isDisabled())throw Error('offline settings save must be disabled')}
    if(route==='/onboarding'){await page.getByRole('radio',{name:'IELTS 7.0'}).click();await page.getByRole('button',{name:'Bắt đầu trận đầu tiên'}).click();await page.getByText('Cần mạng để bắt đầu trận đầu tiên. Kết nối lại rồi thử lại.').waitFor()}
    if(route==='/')await page.getByRole('button',{name:/Đọc hiểu thông báo/}).waitFor();
   }
   if(route==='/roadmap'&&mode==='history-refresh'){
    const historical=page.getByRole('button',{name:/Ngày 2:/});
    await historical.click();await page.getByRole('region',{name:'Nhiệm vụ ngày 2'}).waitFor();
    await historical.evaluate(el=>el.scrollIntoView({block:'center'}));
    const before=await page.evaluate(()=>scrollY);
    const response=page.waitForResponse(res=>new URL(res.url()).pathname==='/api/v1/roadmap');
    await page.evaluate(()=>window.dispatchEvent(new Event('focus')));await response;
    await page.waitForTimeout(100);
    const after=await page.evaluate(()=>scrollY);
    if(Math.abs(after-before)>1||await historical.getAttribute('aria-expanded')!=='true')throw Error(`unchanged roadmap refresh moved historical view ${before} -> ${after}`);
   }
   const slug=route==='/'?'hub':route.slice(1);const filename=`${slug}-${width}-${mode}.png`;
   await page.screenshot({path:`${shots}/${filename}`,fullPage:true});
   const metrics=await page.evaluate(()=>({
    inner:innerWidth,scroll:document.documentElement.scrollWidth,bodyScroll:document.body.scrollWidth,
    zoom:getComputedStyle(document.documentElement).zoom,dpr:devicePixelRatio,
    focus:document.activeElement?.tagName,
    controlOverhang:[...document.querySelectorAll('button,input,a')].filter(el=>{
      const rect=el.getBoundingClientRect();return rect.width>0&&(rect.left < -1 || rect.right > innerWidth+1)
    }).map(el=>el.getAttribute('aria-label')||el.textContent?.trim().slice(0,50)),
   }));
   if(metrics.scroll>metrics.inner+1||metrics.bodyScroll>metrics.inner+1||metrics.controlOverhang.length)throw Error(`${filename} overflow ${JSON.stringify(metrics)}`);
   if(route==='/'&&(mode==='normal'||mode==='zoom200')&&width===375){
    await page.evaluate(()=>window.scrollTo(0,document.documentElement.scrollHeight));
    const visible=await page.evaluate(()=>{
      const task=document.querySelector('ul[aria-label="Nhiệm vụ hôm nay"] > li:last-child')?.getBoundingClientRect();
      const bar=document.querySelector('[data-next-task]')?.getBoundingClientRect();
      return !!task&&!!bar&&task.bottom<=bar.top+1
    });
    if(!visible)throw Error(`${filename} final quest hidden under the bottom action`);
   }
   if(route==='/settings'&&mode.startsWith('confirm')){
    const trap=await page.evaluate(()=>document.querySelector('[role="dialog"]')?.contains(document.activeElement));
    if(!trap)throw Error(`${filename} dialog did not take keyboard focus`);
    await page.keyboard.press('Shift+Tab');
    if(!await page.evaluate(()=>document.activeElement?.textContent?.includes('Đăng xuất')))throw Error(`${filename} dialog did not wrap keyboard focus`);
    await page.keyboard.press('Escape');
    if(await page.getByRole('dialog').count())throw Error(`${filename} Escape did not close confirmation`);
    if(!await page.getByRole('button',{name:'Rời quán · Đăng xuất'}).evaluate(el=>el===document.activeElement))throw Error(`${filename} dialog did not restore opener focus`);
   }
   manifest.push({route,viewport:width,mode,file:filename,metrics});
  }finally{await context.close()}
 }
 for(const width of [375,1280]) for(const route of ['/login','/onboarding','/','/roadmap','/revive','/settings']) await shot(route,width,route==='/login'?'login':'normal');
 for(const [route,mode] of [['/onboarding','quiz'],['/settings','confirm'],['/settings','google-error'],['/roadmap','history-refresh'],['/login','login-error']]) await shot(route,375,mode);
 for(const route of ['/login','/onboarding','/','/roadmap','/revive','/settings']) await shot(route,375,route==='/login'?'login-keyboard':'keyboard');
 for(const route of ['/onboarding','/','/roadmap','/revive','/settings']) for(const mode of ['loading','error','offline']) await shot(route,375,mode);
 for(const route of ['/login','/onboarding','/','/roadmap','/revive','/settings']) await shot(route,375,route==='/login'?'login-zoom200':'zoom200');
 await shot('/settings',375,'confirm-zoom200');
 writeFileSync(`${root}/visual-manifest.json`,JSON.stringify(manifest,null,2));console.log(JSON.stringify({screenshots:manifest.length,zoomScreens:manifest.filter(entry=>entry.metrics.zoom==='2').length,overflow:false}));
}finally{if(browser)await browser.close();try{process.kill(-server.pid,'SIGTERM')}catch{ /* The owned server already exited. */ }await new Promise(r=>setTimeout(r,1200));log.end()}
