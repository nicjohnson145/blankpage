import {create} from '@bufbuild/protobuf';
import {redirect} from "react-router";
import {pauthClient} from "../api/client";
import {IsKeyActiveRequestSchema} from '../gen/pauth/v1beta1/service_pb';
import {ConnectError} from "@connectrpc/connect";
import { LOCAL_STORAGE_ACCESS_KEY_KEY } from '../constants';
import { createContext } from 'react-router';


const accesKeyContext = createContext<string>("");

//type AuthMiddlewareProps = {
//    context: Readonly<RouterContextProvider>;
//}

async function authMiddleware() {
    const key = localStorage.getItem(LOCAL_STORAGE_ACCESS_KEY_KEY)
    if (key === null) {
        throw redirect("/login");
    }

    try {
        const resp = await pauthClient().isKeyActive(create(IsKeyActiveRequestSchema, {
            accessKey: key,
        }))
        if (!resp.active) {
            throw redirect("/login");
        }
    } catch (err) {
        const connectError = ConnectError.from(err);
        console.log(`error checking key validity: ${connectError.message}`)
        throw redirect("/login");
    }
}

export {
    authMiddleware,
    accesKeyContext,
};

