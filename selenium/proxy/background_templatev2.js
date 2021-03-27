
function getRandomInt(max) {
  return Math.floor(Math.random() * Math.floor(max));
}

function getRndProxy() {
  var proxies = [{PROXYIPS}]
  var config = {
    mode: "fixed_servers",
    rules: {
      singleProxy: {
        scheme: "http",
        host: proxies[getRandomInt(proxies.length)],
        port: parseInt({PROXYPORT})
      },
      bypassList: ["foobar.com"]
    }
  };
  console.log("new random proxy set: " + config.rules.singleProxy.host)
  return config
}

chrome.proxy.settings.set({value: getRndProxy(), scope: "regular"}, function() {});

function callbackFn(details) {
    return {
        authCredentials: {
            username: "{PROXYUSER}",
            password: "{PROXYPASS}"
        }
    };
}

chrome.webRequest.onAuthRequired.addListener(
        callbackFn,
        {urls: ["<all_urls>"]},
        ['blocking']
);

chrome.tabs.onUpdated.addListener(function
  (tabId, changeInfo, tab) {
    chrome.proxy.settings.set({value: getRndProxy(), scope: "regular"}, function() {});
    console.log("he refresh?")
  }
);
