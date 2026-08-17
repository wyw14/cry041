const actor='demo-owner'
export async function api<T>(path:string,init:RequestInit={}):Promise<T>{const response=await fetch(`/api/v1${path}`,{...init,headers:{'Content-Type':'application/json','X-Actor-ID':actor,...init.headers}});if(!response.ok){const body=await response.json().catch(()=>({message:'请求失败'}));throw new Error(body.message)}return response.json() as Promise<T>}
