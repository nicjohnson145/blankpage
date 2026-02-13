import {cache} from 'react';
import {createClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {PAuthService} from '../gen/pauth/v1beta1/service_pb';
import {BlankPageService} from '../gen/blankpage/v1/service_pb';

const transport = createConnectTransport({
    baseUrl: "/"
});

const pauthClient = cache(() => {
    return createClient(PAuthService, transport);
});

const blankPageClient = cache(() => {
    return createClient(BlankPageService, transport);
});

export {
    pauthClient,
    blankPageClient,
}
