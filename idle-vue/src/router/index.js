import { createRouter, createWebHistory } from "vue-router";
import LoginView from "../views/LoginView.vue";
import MainView from "../views/MainView.vue";

const routes = [
    { path: "/", redirect: "/login" },
    { path: "/login", component: LoginView },
    { path: "/main", component: MainView }
];

export default createRouter({
    history: createWebHistory(),
    routes
});
