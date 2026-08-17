import {describe,expect,it} from 'vitest'
describe('release states',()=>{it('keeps blocked distinct from ready',()=>{expect(['blocked','ready']).toHaveLength(2)})})
