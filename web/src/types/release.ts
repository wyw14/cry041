export type ReleaseState='preparing'|'reviewing'|'blocked'|'ready'|'released'|'rolled_back'
export interface Release{ id:string; version_name:string; owner_id:string; risk:'low'|'medium'|'high'; state:ReleaseState; revision:number; template_version:number; updated_at:string }
export interface Dashboard{pending:Release[];calendar:Release[];history:Release[];generated_at:string}
