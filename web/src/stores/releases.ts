import {defineStore} from 'pinia'
import {api} from '../services/api'
import type {Dashboard} from '../types/release'
export const useReleaseStore=defineStore('releases',{state:()=>({dashboard:null as Dashboard|null,loading:false,error:''}),actions:{async load(){this.loading=true;this.error='';try{this.dashboard=await api<Dashboard>('/dashboard')}catch(e){this.error=e instanceof Error?e.message:'加载失败'}finally{this.loading=false}}}})
