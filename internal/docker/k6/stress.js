// k6 ramp stress test for the `micro run` gateway.
//
// Ramps VUs up to 500 while exercising all services:
//   health, greeter, contacts, shop (inventory + orders + notifications),
//   platform (posts + comments + mail + users), users service.
//
// Run inside docker:
//   docker compose --profile run run --rm k6 run /scripts/stress.js
// Or standalone against a local `micro run`:
//   BASE_URL=http://localhost:8080 k6 run k6/stress.js

import http from 'k6/http';
import { check, group, sleep, fail } from 'k6';

export const options = {
    stages: [
        { duration: '30s', target: 10 },
        { duration: '30s', target: 50 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 200 },
        { duration: '30s', target: 500 },
        { duration: '1m', target: 500 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<200'],
    },
};

function randomString(length, charset = '') {
    if (!charset) charset = 'abcdefghijklmnopqrstuvwxyz';
    let res = '';
    while (length--) res += charset[(Math.random() * charset.length) | 0];
    return res;
}

const USERNAME = __ENV.MICRO_ADMIN_USER || 'admin';
const PASSWORD = __ENV.MICRO_ADMIN_PASSWORD || 'micro';
const BASE_URL = __ENV.BASE_URL || 'http://micro-run:8080';

export function setup() {
    const res = http.post(
        `${BASE_URL}/auth/login`,
        { id: USERNAME, password: PASSWORD },
        { redirects: 0 }
    );

    check(res, { 'logged in successfully': (r) => r.status === 303 });

    const authToken = res.cookies.micro_token && res.cookies.micro_token[0].value;
    check(authToken, { 'got auth token': () => authToken && authToken.length > 0 });
    if (!authToken) fail('no micro_token cookie returned by /auth/login');

    return authToken;
}

export default function (authToken) {
    const headers = (tag) => ({
        headers: {
            Authorization: `Bearer ${authToken}`,
            'Content-Type': 'application/json',
        },
        tags: Object.assign({}, { name: 'MicroRunStress' }, tag),
    });

    // ── Health + Greeter ───────────────────────────────────────────────────

    group('health + greet', () => {
        const healthRes = http.get(`${BASE_URL}/health`, headers({ name: 'Health' }));
        check(healthRes, { 'gateway healthy': (r) => r.status === 200 });

        const name = `k6-${randomString(5)}`;
        const res = http.post(
            `${BASE_URL}/api/greeter/Greeter.SayHello`,
            JSON.stringify({ name }),
            headers({ name: 'SayHello' })
        );
        check(res, {
            'greet status 200': (r) => r.status === 200,
            'greeting contains name': () => (res.json('message') || '').indexOf(name) !== -1,
        });
    });

    // ── Contacts lifecycle ─────────────────────────────────────────────────

    group('contact lifecycle', () => {
        const createRes = http.post(
            `${BASE_URL}/api/contacts/Contacts.Create`,
            JSON.stringify({
                name: `Stress ${randomString(6)}`,
                email: `${randomString(10)}@example.com`,
                role: 'Tester',
            }),
            headers({ name: 'Create' })
        );
        if (!check(createRes, { 'contact created': (r) => r.status === 200 })) return;

        const contactID = createRes.json('contact.id');

        http.post(`${BASE_URL}/api/contacts/Contacts.Get`,
            JSON.stringify({ id: contactID }), headers({ name: 'Get' }));

        http.post(`${BASE_URL}/api/contacts/Contacts.List`,
            '{}', headers({ name: 'List' }));

        http.post(`${BASE_URL}/api/contacts/Contacts.Update`,
            JSON.stringify({ id: contactID, role: 'Senior Tester' }), headers({ name: 'Update' }));

        http.post(`${BASE_URL}/api/contacts/Contacts.Search`,
            JSON.stringify({ query: 'tester' }), headers({ name: 'Search' }));

        const delRes = http.post(`${BASE_URL}/api/contacts/Contacts.Delete`,
            JSON.stringify({ id: contactID }), headers({ name: 'Delete' }));
        check(delRes, { 'contact deleted': (r) => r.status === 200 });
    });

    // ── Platform.Users ─────────────────────────────────────────────────────

    group('platform users', () => {
        http.post(`${BASE_URL}/api/platform/Users.Signup`,
            JSON.stringify({ name: `k6u-${randomString(6)}`, password: 'pass1234' }),
            headers({ name: 'Signup' }));

        http.post(`${BASE_URL}/api/platform/Users.Login`,
            JSON.stringify({ name: 'alice', password: 'secret123' }),
            headers({ name: 'Login' }));

        http.post(`${BASE_URL}/api/platform/Users.List`,
            '{}', headers({ name: 'UsersList' }));

        http.post(`${BASE_URL}/api/platform/Users.GetProfile`,
            JSON.stringify({ id: 'user-1' }), headers({ name: 'GetProfile' }));

        http.post(`${BASE_URL}/api/platform/Users.UpdateStatus`,
            JSON.stringify({ id: 'user-1', status: `online ${randomString(4)}` }),
            headers({ name: 'UpdateStatus' }));
    });

    // ── Platform.Posts lifecycle ───────────────────────────────────────────

    group('posts lifecycle', () => {
        const createRes = http.post(
            `${BASE_URL}/api/platform/Posts.Create`,
            JSON.stringify({
                title: `Stress Post ${randomString(6)}`,
                content: '# Stress test post',
                author_id: 'user-1',
                author_name: 'alice',
            }),
            headers({ name: 'PostsCreate' })
        );
        if (!check(createRes, { 'post created': (r) => r.status === 200 })) return;

        const postID = createRes.json('post.id');

        http.post(`${BASE_URL}/api/platform/Posts.List`,
            '{}', headers({ name: 'PostsList' }));

        http.post(`${BASE_URL}/api/platform/Posts.Read`,
            JSON.stringify({ id: postID }), headers({ name: 'PostsRead' }));

        http.post(`${BASE_URL}/api/platform/Posts.Update`,
            JSON.stringify({ id: postID, title: 'Updated' }),
            headers({ name: 'PostsUpdate' }));

        http.post(`${BASE_URL}/api/platform/Posts.TagPost`,
            JSON.stringify({ post_id: postID, tag: 'stress' }),
            headers({ name: 'TagPost' }));

        http.post(`${BASE_URL}/api/platform/Posts.ListTags`,
            '{}', headers({ name: 'ListTags' }));

        http.post(`${BASE_URL}/api/platform/Posts.UntagPost`,
            JSON.stringify({ post_id: postID, tag: 'stress' }),
            headers({ name: 'UntagPost' }));

        http.post(`${BASE_URL}/api/platform/Posts.Delete`,
            JSON.stringify({ id: postID }), headers({ name: 'PostsDelete' }));
    });

    // ── Platform.Comments lifecycle ─────────────────────────────────────────

    group('comments lifecycle', () => {
        const createRes = http.post(
            `${BASE_URL}/api/platform/Comments.Create`,
            JSON.stringify({
                post_id: 'post-1',
                content: `Stress comment ${randomString(8)}`,
                author_id: 'user-1',
                author_name: 'alice',
            }),
            headers({ name: 'CommentsCreate' })
        );

        http.post(`${BASE_URL}/api/platform/Comments.List`,
            JSON.stringify({ post_id: 'post-1' }), headers({ name: 'CommentsList' }));

        if (check(createRes, { 'comment created': (r) => r.status === 200 })) {
            http.post(`${BASE_URL}/api/platform/Comments.Delete`,
                JSON.stringify({ id: createRes.json('comment.id') }),
                headers({ name: 'CommentsDelete' }));
        }
    });

    // ── Platform.Mail ──────────────────────────────────────────────────────

    group('mail', () => {
        http.post(`${BASE_URL}/api/platform/Mail.Send`,
            JSON.stringify({ from: 'alice', to: 'bob', subject: `k6 ${randomString(4)}`, body: 'hi' }),
            headers({ name: 'MailSend' }));

        http.post(`${BASE_URL}/api/platform/Mail.Read`,
            JSON.stringify({ user: 'bob' }),
            headers({ name: 'MailRead' }));
    });

    // ── Shop ───────────────────────────────────────────────────────────────

    const sku = 'PHONE-001';

    group('shop lifecycle', () => {
        http.post(`${BASE_URL}/api/shop/InventoryService.Search`,
            JSON.stringify({ query: 'sku' }), headers({ name: 'InvSearch' }));

        http.post(`${BASE_URL}/api/shop/InventoryService.CheckStock`,
            JSON.stringify({ sku }), headers({ name: 'CheckStock' }));

        const reserveRes = http.post(`${BASE_URL}/api/shop/InventoryService.ReserveStock`,
            JSON.stringify({ sku, quantity: 1 }), headers({ name: 'ReserveStock' }));
        check(reserveRes, { 'stock reserved': (r) => r.status === 200 });

        http.post(`${BASE_URL}/api/shop/OrderService.PlaceOrder`,
            JSON.stringify({ sku, customer: 'k6-stress', quantity: 1 }),
            headers({ name: 'PlaceOrder' }));

        http.post(`${BASE_URL}/api/shop/OrderService.ListOrders`,
            JSON.stringify({ customer: 'k6-stress' }),
            headers({ name: 'ListOrders' }));

        http.post(`${BASE_URL}/api/shop/NotificationService.Send`,
            JSON.stringify({ recipient: 'k6-stress', subject: 'Order placed', body: 'Done.', channel: 'email' }),
            headers({ name: 'NotifSend' }));

        http.post(`${BASE_URL}/api/shop/NotificationService.List`,
            JSON.stringify({ recipient: 'k6-stress' }),
            headers({ name: 'NotifList' }));
    });

    // ── Users service ──────────────────────────────────────────────────────

    group('users service', () => {
        const createRes = http.post(
            `${BASE_URL}/api/users/Users.CreateUser`,
            JSON.stringify({ name: `k6user-${randomString(6)}`, email: `${randomString(8)}@stress.com` }),
            headers({ name: 'CreateUser' })
        );

        if (check(createRes, { 'user created': (r) => r.status === 200 })) {
            http.post(`${BASE_URL}/api/users/Users.GetUser`,
                JSON.stringify({ id: createRes.json('user.id') }),
                headers({ name: 'GetUser' }));
        }
    });

    sleep(0.1);
}
