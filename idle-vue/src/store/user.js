import { defineStore } from "pinia";

export const useUserStore = defineStore("user", {
    state: () => ({
        token: "",
        username: "",
    }),
    actions: {
        setUser(name, token) {
            this.username = name;
            this.token = token;
        },
        logout() {
            this.username = "";
            this.token = "";
        }
    }
});
