<script>

const config = {
    pixelName: "Nome_do_Pixel", // Nome que você configurou no arquivo Pixel.php
    //     apiURL: "https://api.SeuDominio.com.br", // URL da sua API (subdomínio da API)
    origin: "utm_source", // O padrão é "utm_source", caso use Hotmart use "xcod"
    
    // ==============================================
    // INFORMAÇÕES DO PRODUTO
    // ==============================================
    isProduct: true, // true = É um infoproduto/encapsulado, false = não é um infoproduto/encapsulado
    productValue: [ 97, 297.90 ], // Valor do seu infoproduto e caso tenha mais de uma oferta/planos colocar assim: [ 9.90, 19.90 ]
   // Se não houver valor ou algo do tipo, pode deixar como productValue : " ",
   productCurrency: "BRL", // Qual o tipo da moeda (ex: "BRL", "USD")
    productCheckout: "checkout", // alguma palavra que contém na URL do seu checkout (Exemplo: 'pay' e 'checkout')

    // ==============================================
    // INFORMAÇÕES DE EVENTOS
    // ==============================================
    sendViewContent: true, // Deseja enviar o evento de ViewContent? Use "true" para enviar ou "false" para não enviar
    sendAddToWishlist: true,
    sendAddToCart: true,
    sendScrolls: false,
    sendTimer_1min: false,
    sendContact: false, // Usar em botões que redirecionam para o WhatsApp. Necessário adicionar a CLASSE no seu botão com o valor de "Contact"
};
const pageRules = [ 
    // ==============================================
    // REGRAS DE PÁGINAS (ENVIAR EVENTOS QUANDO ABRIR PÁGINAS ESPECÍFICAS)
    // ==============================================
    { page: "Pagina_1", event: "Lead", value: 47.90, currency: "BRL" }, // Se na URL existir "Pagina_1" enviar o evento de "Lead"
    { page: "Pagina_2", event: "Lead", value: 97.90, currency: "BRL" },
];
    // ==============================================
    // FORMULÁRIOS (NOME DOS ATRIBUTOS)
    // ==============================================
const forms = {
    email: [ "email", "form_fields[email]" ],
    phone: [ "tel", "telefone", "phone", "form_fields[phone]" ],
    firstName: [ "nome", "name", "form_fields[name]", "form_fields[firstname]" ],
    lastName: [ "sobrenome", "lastname", "form_fields[lastname]" ]
};
// ==================================================

function getCookie(name) {
    const cookies = document.cookie.split(';');
    for (let cookie of cookies) {
        const [key, value] = cookie.trim().split('=');
        if (key === name) {
            return value;
        }
    }
    return null;
}

function getLongExpires() {
    const date = new Date();
    date.setFullYear(date.getFullYear() + 100);
    return `expires=${date.toUTCString()}; path=/`;
}

function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0,
                v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

function getUserData() {
  try { return JSON.parse(getCookie('__userData') || '{}'); }
  catch { return {}; }
}

const eventSent = {};

async function sendEvent(eventType, data = {}) {
    const eventSent = window.__eventSent || (window.__eventSent = {});
    if (["ViewContent", "AddToWishlist", "AddToCart", "InitiateCheckout", "Contact", "Scroll_25", "Scroll_50", "Scroll_75", "Scroll_90", "Timer_1min"].includes(eventType)) {
        if (eventSent[eventType]) return;
        eventSent[eventType] = true;
    }

    const pixelName = config.pixelName;
    const contentId = "";
    const eventId = generateUUID();

    const customEvents = ['Scroll_25', 'Scroll_50', 'Scroll_75', 'Scroll_90', 'Timer_1min', 'PlayVideo', 'ViewVideo_25', 'ViewVideo_50', 'ViewVideo_75', 'ViewVideo_90'];
    const specialEvents = ['AddToCart', 'Lead', 'InitiateCheckout', 'Purchase'];

    if (specialEvents.includes(eventType)) updateUserDataFromForm();

    const _customData = {
        content_name: pixelName,
        ...(contentId != null && contentId !== '' ? { content_ids: [contentId] } : {}),
        content_type: 'product',
        ...(specialEvents.includes(eventType) && data.value !== undefined && data.value > 0 && {
            value: data.value,
            currency: data.currency || config.productCurrency
        })
    };

    const _userData = { eventID: eventId, ...userData };

    if (eventType !== "Init" && eventType !== "UpdateUserData" && !customEvents.includes(eventType)) {
        fbq('track', eventType, _customData, _userData);
    } else if (eventType !== "Init" && eventType !== "UpdateUserData") {
        fbq('trackCustom', eventType, _customData, _userData);
    }

    try {
        const URL = window.location.href;
        let utmParams = {};
        if (eventType === "PageView") {
            const urlParams = new URLSearchParams(window.location.search);
            utmParams = {
                utm_source: urlParams.get('utm_source') || null,
                utm_medium: urlParams.get('utm_medium') || null,
                utm_campaign: urlParams.get('utm_campaign') || null,
                utm_content: urlParams.get('utm_content') || null,
                utm_term: urlParams.get('utm_term') || null
            };
        }
        const __userData = getUserData();
        const payload = { userId, pixelName, contentId, eventType, eventId, URL,
            price: data.value || null,
            currency: data.currency || config.productCurrency,
            fbc: data.fbc || userData.fbc || null,
            fbp: data.fbp || userData.fbp || null,
			first_name: userData.first_name || __userData.first_name || null,
			last_name: userData.last_name || __userData.last_name || null,
			email: userData.email || __userData.email || null,
			phone: userData.phone || __userData.phone || null,
            ...utmParams
        };
        const response = await fetch(config.apiURL + '/events/send', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload)
        });
        if (eventType == "Init") {
            const responseData = await response.json();
            return responseData;
        }
    } catch (e) {
        console.error(`Erro ao rastrear evento de ${eventType}:`, e);
    }
}

function sendEventsByURLMatch() {
  const url = window.location.href.toLowerCase();
  const hostname = window.location.hostname.toLowerCase();

  pageRules.forEach(rule => {
    if (!rule.page || !rule.event) return;
    if (url.includes(rule.page) || hostname.includes(rule.page)) {
        const eventData = {};
        if (rule.value !== undefined) {
            eventData.value = parseFloat(rule.value).toFixed(2);
            eventData.currency = rule.currency || config.productCurrency;
        }
        sendEvent(rule.event, eventData);
    }
  });
}

function updateUserDataFromForm() {
    let first_name, last_name, email, phone;
    const inputs = document.querySelectorAll('input, textarea');

    inputs.forEach(input => {
        const nameAttr = input.name?.toLowerCase();
        const value = input.value?.trim();
        if (!value || !nameAttr) return;

        if (!email && forms.email.some(attr => nameAttr.includes(attr))) {
            email = value.toLowerCase();
        } else if (!phone && forms.phone.some(attr => nameAttr.includes(attr))) {
            phone = value.replace(/\s|-/g, '');
        } else if (!first_name && forms.firstName.some(attr => nameAttr.includes(attr))) {
            const nameParts = value.toLowerCase().split(' ');
            first_name = nameParts[0];
            if (!last_name && nameParts.length > 1) {
                last_name = nameParts[nameParts.length - 1];
            }
        } else if (!last_name && forms.lastName.some(attr => nameAttr.includes(attr))) {
            last_name = value.toLowerCase();
        }
    });

    let userData = {};
    const current = getCookie("__userData");
    if (current) {
        try {
            userData = JSON.parse(current);
        } catch (e) {
            userData = {};
        }
    }

    if (first_name) userData.first_name = first_name;
    if (last_name) userData.last_name = last_name;
    if (email) userData.email = email;
    if (phone) userData.phone = phone;

    const expires = getLongExpires();
    document.cookie = "__userData=" + JSON.stringify(userData) + "; " + expires;
}

let userData = {};
let [fbc, fbp] = ["_fbc", "_fbp"].map(getCookie);
let { userId, first_name, last_name, email, phone } = getUserData();

if (!userId) {
    userId = generateUUID();
    const date = new Date();
    date.setFullYear(date.getFullYear() + 100); 
    const current = getCookie("__userData");
    if (current) {
        try { userData = JSON.parse(current); }
        catch (e) { userData = {}; }
    }
    userData.userId = userId;
    document.cookie = `__userData=${JSON.stringify(userData)}; expires=${date.toUTCString()}; path=/`;
}
	
(async () => {
    const data = await sendEvent('Init') || {};

    userData = {
        ...(data.city && { ct: data.city }),
        ...(data.state && { st: data.state }),
        ...(data.zipCode && { zp: data.zipCode }),
        ...(data.country && { country: data.country }),
        ...(data.client_ip_address && { client_ip_address: data.client_ip_address }),
        ...(data.client_user_agent && { client_user_agent: data.client_user_agent }),
        ...(data.fbc || fbc ? { fbc: data.fbc || fbc } : {}),
        ...(data.fbp || fbp ? { fbp: data.fbp || fbp } : {}),
        ...(userId && { external_id: userId }),
        ...(first_name && { fn: first_name }),
        ...(last_name && { ln: last_name }),
        ...(email && { em: email }),
        ...(phone && { ph: phone })
    };

    !function(f,b,e,v,n,t,s)
    {if(f.fbq)return;n=f.fbq=function(){n.callMethod?
    n.callMethod.apply(n,arguments):n.queue.push(arguments)};
    if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
    n.queue=[];t=b.createElement(e);t.async=!0;
    t.src=v;s=b.getElementsByTagName(e)[0];
    s.parentNode.insertBefore(t,s)}(window, document,'script',
    'https://connect.facebook.net/en_US/fbevents.js');
    fbq('init', 'SEU_PIXEL_ID_1', userData);

    await waitForCookiesAndSendEvent();

    setUTMs();
    setButtonsURLs();
    setFormListeners();

    const observeEventByIdOrClass = (name, callback) => {
        const checkAndObserve = () => {
            const elements = [
                ...document.querySelectorAll(`#${name}`),
                ...document.querySelectorAll(`.${name}`)
            ];
            elements.forEach(el => {
                if (!el.dataset.observed) {
                    callback(el);
                    el.dataset.observed = "true";
                }
            });
        };

        checkAndObserve();

        const observer = new MutationObserver(checkAndObserve);
        observer.observe(document.body, { childList: true, subtree: true });
    };

    const setupIntersectionEvent = (eventName) => {
        observeEventByIdOrClass(eventName, (el) => {
            const io = new IntersectionObserver((entries, ioInstance) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting && !eventSent[eventName]) {
                        sendEvent(eventName);
                        eventSent[eventName] = true;
                        ioInstance.unobserve(entry.target);
                    }
                });
            });
            io.observe(el);
        });
    };

    if (config.sendViewContent) {
        setupIntersectionEvent("ViewContent");
    }

    if (config.sendAddToWishlist) {
        setupIntersectionEvent("AddToWishlist");
    }

    if (config.sendAddToCart) {
        observeEventByIdOrClass("AddToCart", (el) => {
            el.addEventListener('click', () => {
                if (!eventSent["AddToCart"]) {
                    sendEvent("AddToCart");
                    eventSent["AddToCart"] = true;
                }
            });
        });
    }

    if (config.sendScrolls) {
        (function () {
            let scrollEvents = [25, 50, 75, 90];

            function trackScrollEvent() {
                let scrollPosition = window.scrollY + window.innerHeight;
                let pageHeight = document.documentElement.scrollHeight;
                let scrollPercentage = (scrollPosition / pageHeight) * 100;

                scrollEvents.forEach(percent => {
                    if (scrollPercentage >= percent) {
                        sendEvent(`Scroll_${percent}`);
                    }
                });
            }

            window.addEventListener('scroll', trackScrollEvent);
        })();
    }

    if (config.sendTimer_1min) {
        setTimeout(() => {
            sendEvent("Timer_1min");
        }, 60000);
    }

    if (config.sendContact) {
        observeEventByIdOrClass("Contact", (el) => {
            el.addEventListener('click', () => {
                if (!eventSent["Contact"]) {
                    sendEvent("Contact");
                    eventSent["Contact"] = true;
                }
            });
        });
    }

})();

async function instantPixel() {
    const urlParams = new URLSearchParams(window.location.search);
    const hasFbclid = urlParams.has('fbclid');
    const currentHostname = window.location.hostname;

    sendEvent('PageView');
    sendEventsByURLMatch();

    if (!hasFbclid) {
        const fbc = getCookie('_fbc');
        const fbp = getCookie('_fbp');

        userData = {
            ...userData,
            ...(fbc ? { fbc } : {}),
            ...(fbp ? { fbp } : {})
        };

        await sendEvent('UpdateUserData');
        return;
    } else {
        const checkInterval = 100;
        const maxWaitTime = 5000;
        let waited = 0;

        const interval = setInterval(async () => {
            const fbc = getCookie('_fbc');
            const fbp = getCookie('_fbp');

            const hasFbc = fbc && fbc.trim() !== "";
            const hasFbp = fbp && fbp.trim() !== "";

            if (hasFbc && hasFbp) {
                clearInterval(interval);

                userData = {
                    ...userData,
                    ...(fbc ? { fbc } : {}),
                    ...(fbp ? { fbp } : {})
                };

                await sendEvent('UpdateUserData');
            } else {
                waited += checkInterval;
                if (waited >= maxWaitTime) {
                    clearInterval(interval);                
                    const fbc_2 = getCookie('_fbc');
                    const fbp_2 = getCookie('_fbp');

                    console.warn("Cookies _fbc e _fbp não foram encontrados dentro do tempo esperado.");
                    userData = {
                        ...userData,
                        ...(fbc_2 ? { fbc: fbc_2 } : {}),
                        ...(fbp_2 ? { fbp: fbp_2 } : {})
                    };

                    await sendEvent('UpdateUserData');
                }
            }
        }, checkInterval);
    }
}

async function waitForCookiesAndSendEvent() {
    const urlParams = new URLSearchParams(window.location.search);
    const hasFbclid = urlParams.has('fbclid');

    fbq('trackCustom', 'createCookies');

    if (!hasFbclid) {
        const checkInterval = 100;
        const maxWaitTime = 5000;
        let waited = 0;

        const interval = setInterval(async () => {
            const fbc = getCookie('_fbc');
            const fbp = getCookie('_fbp');

            const hasFbp = fbp && fbp.trim() !== "";

            if (hasFbp) {
                clearInterval(interval);

                userData = {
                    ...userData,
                    ...(fbc ? { fbc } : {}),
                    ...(fbp ? { fbp } : {})
                };

                await sendEvent('PageView');
                sendEventsByURLMatch();
            } else {
                waited += checkInterval;

                if (waited >= maxWaitTime) {
                    clearInterval(interval);

                    const fbc_2 = getCookie('_fbc');
                    const fbp_2 = getCookie('_fbp');

                    console.warn("Cookie _fbp não foi encontrado dentro do tempo esperado.");

                    userData = {
                        ...userData,
                        ...(fbc_2 ? { fbc: fbc_2 } : {}),
                        ...(fbp_2 ? { fbp: fbp_2 } : {})
                    };

                    await sendEvent('PageView');
                    sendEventsByURLMatch();
                }
            }
        }, checkInterval);
    } else {
        const checkInterval = 100;
        const maxWaitTime = 5000;
        let waited = 0;

        const interval = setInterval(async () => {
            const fbc = getCookie('_fbc');
            const fbp = getCookie('_fbp');

            const hasFbc = fbc && fbc.trim() !== "";
            const hasFbp = fbp && fbp.trim() !== "";

            if (hasFbc && hasFbp) {
                clearInterval(interval);

                userData = {
                    ...userData,
                    ...(fbc ? { fbc } : {}),
                    ...(fbp ? { fbp } : {})
                };

                await sendEvent('PageView');
                sendEventsByURLMatch();
            } else {
                waited += checkInterval;
                if (waited >= maxWaitTime) {
                    clearInterval(interval);                
                    const fbc_2 = getCookie('_fbc');
                    const fbp_2 = getCookie('_fbp');

                    console.warn("Cookies _fbc e _fbp não foram encontrados dentro do tempo esperado.");
                    userData = {
                        ...userData,
                        ...(fbc_2 ? { fbc: fbc_2 } : {}),
                        ...(fbp_2 ? { fbp: fbp_2 } : {})
                    };

                    await sendEvent('PageView');
                    sendEventsByURLMatch();
                }
            }
        }, checkInterval);
    }
}

function setUTMs() {
    const url = new URL(window.location.href);
    let UTMs = url.searchParams.get(config.origin);

    if (!UTMs) {
        url.searchParams.set(config.origin, "|userId" + userId + "|");
        console.log('UTMs criado:', "|userId" + userId + "|");
    } else if (!UTMs.includes("|userId" + userId + "|")) {
        url.searchParams.set(config.origin, `${UTMs}|userId${userId}|`);
        console.log('UTMs atualizado:', `${UTMs}|userId${userId}|`);
    }

    window.history.replaceState({}, '', url);
}

function setButtonsURLs() {
    if (!config.isProduct) return;
    
    const buttons = Array.from(document.querySelectorAll('a[href]'))
    .filter(link => {
        const href = link.getAttribute('href')?.toLowerCase();
        if (!href || href.startsWith('#') || href.startsWith('javascript:')) return false;
        return href.includes(config.productCheckout);
    });
    
    console.log(buttons);
    buttons.forEach(link => {
        try {
            const url = new URL(link.href);
            if (url.searchParams.has(config.origin)) {
                if (url.searchParams.get(config.origin) !== "|userId" + userId + "|") {
                    url.searchParams.set(config.origin, "|userId" + userId + "|");
                }
            } else {
                url.searchParams.append(config.origin, "|userId" + userId + "|");
            }
            link.href = url.toString();
        } catch (e) {
            console.warn('Link inválido:', link.href);
        }
    });

    if (config.productValue.length === buttons.length) {
        buttons.forEach((button, index) => {
            console.log('Botão ' + index);
            const value = config.productValue[index];
            if (!button.dataset.price) {
                button.addEventListener('click', () => {
                    sendEvent('InitiateCheckout', {
                        value: value,
                        currency: config.productCurrency
                    });
                });
                button.dataset.price = Number(value).toFixed(2);
            }
        });
    } else {
        const observer = new MutationObserver(() => {
            const buttons = Array.from(document.querySelectorAll('a[href]'))
                .filter(link => {
                    const href = link.getAttribute('href')?.toLowerCase();
                    if (!href || href.startsWith('#') || href.startsWith('javascript:')) return false;
                    return href.includes(config.productCheckout);
                });

            buttons.forEach((link, index) => {
                const value = config.productValue[index];

                if (link.dataset.price) return;

                try {
                    const url = new URL(link.href);
                    if (url.searchParams.has(config.origin)) {
                        if (url.searchParams.get(config.origin) !== "|userId" + userId + "|") {
                            url.searchParams.set(config.origin, "|userId" + userId + "|");
                        }
                    } else {
                        url.searchParams.append(config.origin, "|userId" + userId + "|");
                    }
                    link.href = url.toString();
                } catch (e) {
                    console.warn('Link inválido:', link.href);
                }

                link.addEventListener('click', () => {
                    sendEvent('InitiateCheckout', {
                        value: value,
                        currency: config.productCurrency
                    });
                });
                link.dataset.price = value;
            });

            if (buttons.length === config.productValue.length) {
                console.log("✅ Todos os botões foram encontrados. Desligando observer.");
                observer.disconnect();
            }
        });
        observer.observe(document.body, { childList: true, subtree: true });
    }
}

function setFormListeners() {
    const attachListener = (form) => {
        if (form.dataset.leadSet) return;

        const __userData = getUserData();
        const userId = __userData.userId;

        if (!userId) {
            console.log("⚠️ Nenhum userId encontrado no cookie.");
        } else {
            const fields = form.querySelectorAll('[name*="utm_source"]');

            if (fields.length > 0) {
                fields.forEach(field => {
                    field.value = "|userId" + userId + "|";
                    console.log("✅ userId aplicado no campo existente:", field.name, userId);
                });
            } else {
                const hiddenField = document.createElement("input");
                hiddenField.type = "hidden";
                hiddenField.name = "utm_source";
                hiddenField.value = "|userId" + userId + "|";
                form.appendChild(hiddenField);

                console.log("✅ Campo utm_source criado e userId aplicado:", userId);
            }
        }

        form.addEventListener('submit', () => {
            sendEvent('Lead');
        });
        form.dataset.leadSet = "true";
    };

    const forms = document.querySelectorAll('form');
    forms.forEach(attachListener);

    const observer = new MutationObserver(() => {
        const newForms = document.querySelectorAll('form:not([data-lead-set])');
        newForms.forEach(attachListener);
    });

    observer.observe(document.body, { childList: true, subtree: true });
}
</script>

