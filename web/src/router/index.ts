import {createRouter,createWebHistory} from 'vue-router'
import ReleaseList from '../pages/ReleaseList.vue'
import Workspace from '../pages/Workspace.vue'
import Templates from '../pages/Templates.vue'
import Snapshots from '../pages/Snapshots.vue'
export default createRouter({history:createWebHistory(),routes:[{path:'/',component:ReleaseList},{path:'/release/:id',component:Workspace},{path:'/templates',component:Templates},{path:'/snapshots',component:Snapshots}]})
