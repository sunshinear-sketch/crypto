const $=id=>document.getElementById(id);
const money=v=>Number(v).toLocaleString('en-US',{maximumFractionDigits:2});
const num=v=>Number(v).toLocaleString('en-US',{maximumSignificantDigits:7});
const fixed=(v,n=1)=>Number(v).toFixed(n);
async function load(){
 const q=new URLSearchParams({q:$('q').value,maxRsi:$('rsi').value,maxDistance:$('distance').value,minVolume:$('volume').value,limit:'500'});
 $('status').textContent='Loading…';
 try{
  const res=await fetch('/api/markets?'+q);
  if(!res.ok)throw Error('HTTP '+res.status);
  const j=await res.json();
  $('count').textContent=j.count;
  $('updated').textContent=j.updatedAt&&j.updatedAt!=='0001-01-01T00:00:00Z'?new Date(j.updatedAt).toLocaleString():'Waiting for first scan';
  $('status').textContent='Connected';
  $('rows').replaceChildren();
  if(!j.data.length){
   const tr=document.createElement('tr'),td=document.createElement('td');td.colSpan=9;td.textContent='No matching markets yet. Initial scan may take several minutes, or try broader filters.';tr.append(td);$('rows').append(tr);return;
  }
  for(const m of j.data){
   const tr=document.createElement('tr');
   const values=[m.symbol,num(m.price),fixed(m.rsi4h),fixed(m.rsi1d),num(m.low90d),fixed(m.distance90d,2)+'%','$'+money(m.volume24h),fixed(m.fundingRate*100,4)+'%',fixed(m.score,0)];
   for(let i=0;i<values.length;i++){const td=document.createElement('td');td.textContent=values[i];if(i===0)td.className='symbol';if(i===8)td.className='score';tr.append(td)}
   $('rows').append(tr);
  }
 }catch(e){
  $('status').textContent='Connection error';
  $('rows').replaceChildren();
  const tr=document.createElement('tr'),td=document.createElement('td');td.colSpan=9;td.textContent='Could not load data. Check server logs.';tr.append(td);$('rows').append(tr);console.error(e);
 }
}
$('apply').addEventListener('click',load);
$('refresh').addEventListener('click',load);
load();setInterval(load,120000);
