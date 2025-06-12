// firebase-messaging-sw.js
importScripts('https://www.gstatic.com/firebasejs/10.12.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/10.12.0/firebase-messaging-compat.js');

// Initialize Firebase in service worker
firebase.initializeApp({
    apiKey: "AIzaSyAGrmgZPIZeO4yD1ng6RSyRu0GgapNB-YE",
    authDomain: "swptest-7f1bb.firebaseapp.com",
    databaseURL: "https://swptest-7f1bb-default-rtdb.firebaseio.com",
    projectId: "swptest-7f1bb",
    storageBucket: "swptest-7f1bb.appspot.com",
    messagingSenderId: "312264882389",
    appId: "1:312264882389:web:cff6e4f72e3eb201518a5c",
    measurementId: "G-5DFMCPBY4W"
});

const messaging = firebase.messaging();
