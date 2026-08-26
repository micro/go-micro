// k6 load test for the `micro run` gateway (see ../docker-compose.yml).
//
// Coverage: health, greeter, contacts CRUD, shop (inventory + orders + notifications),
//           platform (posts + comments + mail + users), users service.
//
// Run inside docker (profile "run"):
//   docker compose --profile run up k6
// Or standalone against a local `micro run`:
//   BASE_URL=http://localhost:8080 k6 run k6/loadtest.js

import http from 'k6/http';
import { check, group, fail } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 1),
  iterations: Number(__ENV.ITERATIONS || 1),
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
    tags: Object.assign({}, { name: 'MicroRun' }, tag),
  });

  // ── Greeter ──────────────────────────────────────────────────────────────

  group('01. Gateway health', () => {
    const res = http.get(`${BASE_URL}/health`, headers({ name: 'Health' }));
    check(res, { 'gateway healthy': (r) => r.status === 200 });
  });

  group('02. Greeter.SayHello', () => {
    const name = `k6-${randomString(5)}`;
    const res = http.post(
      `${BASE_URL}/api/greeter/Greeter.SayHello`,
      JSON.stringify({ name }),
      headers({ name: 'SayHello' })
    );
    const ok = check(res, {
      'greeter responded': (r) => r.status === 200,
      'greeting contains name': () => (res.json('message') || '').indexOf(name) !== -1,
    });
    if (!ok) { console.log(`Greeter failed ${res.status} ${res.body}`); return; }
  });

  // ── Contacts ─────────────────────────────────────────────────────────────

  let contactID;

  group('03. Contacts.Create', () => {
    const payload = {
      name: `Load Test ${randomString(6)}`,
      email: `${randomString(10)}@example.com`,
      role: 'Tester',
    };
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.Create`,
      JSON.stringify(payload),
      headers({ name: 'Create' })
    );
    if (check(res, { 'contact created': (r) => r.status === 200 })) {
      contactID = res.json('contact.id');
    } else {
      console.log(`Unable to create contact ${res.status} ${res.body}`);
      return;
    }
  });

  if (!contactID) return;

  group('04. Contacts.List', () => {
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.List`,
      '{}',
      headers({ name: 'List' })
    );
    check(res, {
      'list status 200': (r) => r.status === 200,
      'contacts not empty': (r) => (res.json('contacts') || []).length > 0,
    });
  });

  group('05. Contacts.Get', () => {
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.Get`,
      JSON.stringify({ id: contactID }),
      headers({ name: 'Get' })
    );
    check(res, { 'get status 200': (r) => r.status === 200 });
  });

  group('06. Contacts.Update', () => {
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.Update`,
      JSON.stringify({ id: contactID, role: 'Senior Tester' }),
      headers({ name: 'Update' })
    );
    const ok = check(res, {
      'update status 200': (r) => r.status === 200,
      'updated role correct': () => res.json('contact.role') === 'Senior Tester',
    });
    if (!ok) { console.log(`Unable to update contact ${res.status} ${res.body}`); return; }
  });

  group('07. Contacts.Search', () => {
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.Search`,
      JSON.stringify({ query: 'tester' }),
      headers({ name: 'Search' })
    );
    check(res, {
      'search status 200': (r) => r.status === 200,
      'found our contact': (r) =>
        (res.json('contacts') || []).some((c) => c.id === contactID),
    });
  });

  group('08. Contacts.Delete', () => {
    const res = http.post(
      `${BASE_URL}/api/contacts/Contacts.Delete`,
      JSON.stringify({ id: contactID }),
      headers({ name: 'Delete' })
    );
    const ok = check(res, { 'deleted status 200': (r) => r.status === 200 });
    if (!ok) { console.log(`Contact delete failed ${res.status} ${res.body}`); return; }
  });

  // ── Platform.Users ───────────────────────────────────────────────────────

  let platformUserToken;

  group('09. Users.Signup', () => {
    const uname = `k6user-${randomString(6)}`;
    const res = http.post(
      `${BASE_URL}/api/platform/Users.Signup`,
      JSON.stringify({ name: uname, password: 'pass1234' }),
      headers({ name: 'Signup' })
    );
    const ok = check(res, { 'signup status 200': (r) => r.status === 200 });
    if (ok) platformUserToken = res.json('token');
  });

  group('10. Users.Login', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Users.Login`,
      JSON.stringify({ name: 'alice', password: 'secret123' }),
      headers({ name: 'Login' })
    );
    check(res, { 'login status 200': (r) => r.status === 200 });
  });

  group('11. Users.List', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Users.List`,
      '{}',
      headers({ name: 'UsersList' })
    );
    check(res, { 'users list status 200': (r) => r.status === 200 });
  });

  group('12. Users.GetProfile', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Users.GetProfile`,
      JSON.stringify({ id: 'user-1' }),
      headers({ name: 'GetProfile' })
    );
    check(res, { 'profile status 200': (r) => r.status === 200 });
  });

  group('13. Users.UpdateStatus', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Users.UpdateStatus`,
      JSON.stringify({ id: 'user-1', status: `online ${randomString(4)}` }),
      headers({ name: 'UpdateStatus' })
    );
    check(res, { 'update status 200': (r) => r.status === 200 });
  });

  // ── Platform.Posts ───────────────────────────────────────────────────────

  let postID;

  group('14. Posts.Create', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.Create`,
      JSON.stringify({
        title: `Load Test Post ${randomString(6)}`,
        content: '# Hello\nThis is a test post.',
        author_id: 'user-1',
        author_name: 'alice',
      }),
      headers({ name: 'PostsCreate' })
    );
    if (check(res, { 'post created': (r) => r.status === 200 })) {
      postID = res.json('post.id');
    }
  });

  if (!postID) return;

  group('15. Posts.List', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.List`,
      '{}',
      headers({ name: 'PostsList' })
    );
    check(res, {
      'posts list status 200': (r) => r.status === 200,
      'posts not empty': (r) => (res.json('posts') || []).length > 0,
    });
  });

  group('16. Posts.Read', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.Read`,
      JSON.stringify({ id: postID }),
      headers({ name: 'PostsRead' })
    );
    check(res, { 'post read status 200': (r) => r.status === 200 });
  });

  group('17. Posts.Update', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.Update`,
      JSON.stringify({ id: postID, title: 'Updated Title' }),
      headers({ name: 'PostsUpdate' })
    );
    check(res, { 'post updated status 200': (r) => r.status === 200 });
  });

  group('18. Posts.TagPost + UntagPost', () => {
    const tagRes = http.post(
      `${BASE_URL}/api/platform/Posts.TagPost`,
      JSON.stringify({ post_id: postID, tag: 'loadtest' }),
      headers({ name: 'TagPost' })
    );
    check(tagRes, { 'tag status 200': (r) => r.status === 200 });

    const untagRes = http.post(
      `${BASE_URL}/api/platform/Posts.UntagPost`,
      JSON.stringify({ post_id: postID, tag: 'loadtest' }),
      headers({ name: 'UntagPost' })
    );
    check(untagRes, { 'untag status 200': (r) => r.status === 200 });
  });

  group('19. Posts.ListTags', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.ListTags`,
      '{}',
      headers({ name: 'ListTags' })
    );
    check(res, { 'list tags status 200': (r) => r.status === 200 });
  });

  group('20. Posts.Delete', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Posts.Delete`,
      JSON.stringify({ id: postID }),
      headers({ name: 'PostsDelete' })
    );
    check(res, { 'post deleted status 200': (r) => r.status === 200 });
  });

  // ── Platform.Comments ────────────────────────────────────────────────────

  let commentID;

  group('21. Comments.Create', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Comments.Create`,
      JSON.stringify({
        post_id: 'post-1',
        content: `Test comment ${randomString(8)}`,
        author_id: 'user-1',
        author_name: 'alice',
      }),
      headers({ name: 'CommentsCreate' })
    );
    if (check(res, { 'comment created': (r) => r.status === 200 })) {
      commentID = res.json('comment.id');
    }
  });

  group('22. Comments.List', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Comments.List`,
      JSON.stringify({ post_id: 'post-1' }),
      headers({ name: 'CommentsList' })
    );
    check(res, { 'comments list status 200': (r) => r.status === 200 });
  });

  if (commentID) {
    group('23. Comments.Delete', () => {
      const res = http.post(
        `${BASE_URL}/api/platform/Comments.Delete`,
        JSON.stringify({ id: commentID }),
        headers({ name: 'CommentsDelete' })
      );
      check(res, { 'comment deleted status 200': (r) => r.status === 200 });
    });
  }

  // ── Platform.Mail ────────────────────────────────────────────────────────

  group('24. Mail.Send', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Mail.Send`,
      JSON.stringify({
        from: 'alice',
        to: 'bob',
        subject: `k6 test ${randomString(4)}`,
        body: 'Hello from k6',
      }),
      headers({ name: 'MailSend' })
    );
    check(res, { 'mail sent status 200': (r) => r.status === 200 });
  });

  group('25. Mail.Read', () => {
    const res = http.post(
      `${BASE_URL}/api/platform/Mail.Read`,
      JSON.stringify({ user: 'bob' }),
      headers({ name: 'MailRead' })
    );
    check(res, { 'mail read status 200': (r) => r.status === 200 });
  });

  // ── Shop.InventoryService ────────────────────────────────────────────────

  const sku = 'PHONE-001';

  group('26. InventoryService.Search', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/InventoryService.Search`,
      JSON.stringify({ query: 'sku' }),
      headers({ name: 'InvSearch' })
    );
    check(res, { 'inv search status 200': (r) => r.status === 200 });
  });

  group('27. InventoryService.CheckStock', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/InventoryService.CheckStock`,
      JSON.stringify({ sku }),
      headers({ name: 'CheckStock' })
    );
    check(res, { 'check stock status 200': (r) => r.status === 200 });
  });

  group('28. InventoryService.ReserveStock', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/InventoryService.ReserveStock`,
      JSON.stringify({ sku, quantity: 1 }),
      headers({ name: 'ReserveStock' })
    );
    check(res, { 'reserve stock status 200': (r) => r.status === 200 });
  });

  // ── Shop.OrderService ────────────────────────────────────────────────────

  group('29. OrderService.PlaceOrder', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/OrderService.PlaceOrder`,
      JSON.stringify({ sku, customer: 'k6-load', quantity: 1 }),
      headers({ name: 'PlaceOrder' })
    );
    check(res, { 'place order status 200': (r) => r.status === 200 });
  });

  group('30. OrderService.ListOrders', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/OrderService.ListOrders`,
      JSON.stringify({ customer: 'k6-load' }),
      headers({ name: 'ListOrders' })
    );
    check(res, {
      'list orders status 200': (r) => r.status === 200,
      'orders not empty': (r) => (res.json('orders') || []).length > 0,
    });
  });

  group('31. OrderService.GetOrder', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/OrderService.ListOrders`,
      JSON.stringify({ customer: 'k6-load' }),
      headers({ name: 'GetOrder' })
    );
    const orders = res.json('orders') || [];
    if (orders.length > 0) {
      const id = orders[0].id;
      const getRes = http.post(
        `${BASE_URL}/api/shop/OrderService.GetOrder`,
        JSON.stringify({ id }),
        headers({ name: 'GetOrder' })
      );
      check(getRes, { 'get order status 200': (r) => r.status === 200 });
    }
  });

  // ── Shop.NotificationService ─────────────────────────────────────────────

  group('32. NotificationService.Send', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/NotificationService.Send`,
      JSON.stringify({
        recipient: 'k6-load',
        subject: 'Order confirmation',
        body: 'Your order has been placed.',
        channel: 'email',
      }),
      headers({ name: 'NotifSend' })
    );
    check(res, { 'notif sent status 200': (r) => r.status === 200 });
  });

  group('33. NotificationService.List', () => {
    const res = http.post(
      `${BASE_URL}/api/shop/NotificationService.List`,
      JSON.stringify({ recipient: 'k6-load' }),
      headers({ name: 'NotifList' })
    );
    check(res, { 'notif list status 200': (r) => r.status === 200 });
  });

  // ── Users service ────────────────────────────────────────────────────────

  let createdUserID;

  group('34. users.CreateUser', () => {
    const res = http.post(
      `${BASE_URL}/api/users/Users.CreateUser`,
      JSON.stringify({
        name: `k6user-${randomString(6)}`,
        email: `${randomString(8)}@example.com`,
      }),
      headers({ name: 'CreateUser' })
    );
    if (check(res, { 'user created status 200': (r) => r.status === 200 })) {
      createdUserID = res.json('user.id');
    }
  });

  if (createdUserID) {
    group('35. users.GetUser', () => {
      const res = http.post(
        `${BASE_URL}/api/users/Users.GetUser`,
        JSON.stringify({ id: createdUserID }),
        headers({ name: 'GetUser' })
      );
      check(res, { 'get user status 200': (r) => r.status === 200 });
    });
  }
}
