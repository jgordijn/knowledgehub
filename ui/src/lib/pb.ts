import PocketBase from 'pocketbase';
import { authStore } from './auth-store';

const pb = new PocketBase('/', authStore);
export default pb;
