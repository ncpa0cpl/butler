class TabBtn {
  /** @type {TabsElement} */
  parent;
  /** @type {HTMLButtonElement} */
  btn;
  /** @type {string} */
  tabID;

  /**
   * @param {TabsElement} parent
   * @param {HTMLButtonElement} btn
   * @param {string} tabID
   */
  constructor(parent, btn, tabID) {
    this.parent = parent;
    this.btn = btn;
    this.tabID = tabID;
    btn.addEventListener("click", () => {
      parent.openTab(tabID);
    });
  }

  /**
   * @param {bool} active
   */
  setActive(active) {
    const activeClasses = ["bg-white", "text-black", "shadow-sm", "dark:bg-slate-800", "dark:text-gray-100"];
    if (active) {
      this.btn.classList.add(...activeClasses);
    } else {
      this.btn.classList.remove(...activeClasses);
    }
  }
}

class TabContent {
  /** @type {TabsElement} */
  parent;
  /** @type {HTMLElement} */
  elem;
  /** @type {string} */
  tabID;

  /**
   * @param {TabsElement} parent
   * @param {HTMLButtonElement} btn
   * @param {string} tabID
   */
  constructor(parent, elem, tabID) {
    this.parent = parent;
    this.elem = elem;
    this.tabID = tabID;
  }

  hide() {
    this.elem.classList.add("hidden");
  }

  show() {
    this.elem.classList.remove("hidden");
    this.parent.setActiveTabID(this.tabID);
  }
}

class TabsElement extends HTMLDivElement {
  /** @type {TabBtn[]} */
  buttons = [];
  /** @type {Array<TabContent>} */
  contentElems = [];
  activeTab = "";

  /** @param {string} id */
  setActiveTabID(id) {
    this.activeTab = id;
    for (const btn of this.buttons) {
      if (btn.tabID === id) {
        btn.setActive(true);
      } else {
        btn.setActive(false);
      }
    }
  }

  /** @param {string} tabID  */
  openTab(tabID) {
    for (const tab of this.contentElems) {
      if (tab.tabID === tabID) {
        tab.show();
      } else {
        tab.hide();
      }
    }
  }

  connectedCallback() {
    /** @type {NodeListOf<HTMLButtonElement>} */
    const buttons = this.querySelectorAll(".tab-buttons > button");
    for (const btn of buttons) {
      const tabID = btn.dataset.tabid;
      if (tabID) {
        this.buttons.push(new TabBtn(this, btn, tabID));
      }
    }

    const contents = this.querySelectorAll(".tab-content");
    for (const content of contents) {
      const tabID = content.dataset.tabid;
      const btn = this.buttons.find((btn) => btn.tabID === tabID);
      if (btn && tabID) {
        const tab = new TabContent(this, content, tabID);
        this.contentElems.push(tab);
        if (this.activeTab === "") {
          tab.show();
        }
      }
    }
  }
}

customElements.define("cst-tabs", TabsElement, { extends: "div" });

/** @type {GroupBtn[]} */
const allGroupBtns = [];

class GroupBtn extends HTMLButtonElement {
  endpointID = "";
  /** @type {Element} */
  ownDetails;
  /** @type {NodeListOf<Element> | undefined} */
  endpointDetails;
  endpointLabel = "";

  setActive(active) {
    const activeClasses = ["bg-gray-100", "dark:bg-slate-700", "text-primary", "text-black", "dark:text-white"];
    const inactiveClasses = ["text-gray-600", "dark:text-gray-300"];

    if (active) {
      this.classList.remove(...inactiveClasses);
      this.classList.add(...activeClasses);
    } else {
      this.classList.remove(...activeClasses);
      this.classList.add(...inactiveClasses);
    }
  }

  open() {
    if (!this.ownDetails.classList.contains("hidden")) {
      return;
    }

    for (const endpoint of this.endpointDetails) {
      endpoint.classList.add("hidden");
    }
    this.ownDetails.classList.remove("hidden");

    this.setActive(true);
    for (const btn of allGroupBtns) {
      if (btn != this) {
        btn.setActive(false);
      }
    }

    if (this.endpointLabel) {
      document.title = this.endpointLabel;
    }

    let parent = this.parentElement;
    while (parent != null) {
      if (parent.__collapsibleContainer) {
        parent.__collapsibleBtn.open();
      }
      parent = parent.parentElement;
    }
  }

  connectedCallback() {
    const endpId = this.dataset.endpid;
    if (!endpId) return;
    this.endpointID = endpId;

    const elabel = this.dataset.label;
    if (elabel) {
      this.endpointLabel = elabel;
    }

    this.endpointDetails = document.querySelectorAll(".endp-details");
    for (const endpoint of this.endpointDetails) {
      const id = endpoint.dataset.endpid;
      if (id === endpId) {
        this.ownDetails = endpoint;
      }
    }

    this.addEventListener("click", () => {
      this.open();

      const url = new URL(window.location.href);
      url.searchParams.set("routeID", endpId);
      window.history.pushState(undefined, undefined, url);
    });

    allGroupBtns.push(this);
  }
}

setTimeout(() => {
  const initRoute = new URL(window.location.href).searchParams.get("routeID");
  if (initRoute) {
    for (const gButton of allGroupBtns) {
      if (gButton.endpointID === initRoute) {
        gButton.open();
        break;
      }
    }
  }

  window.addEventListener("popstate", () => {
    const routeID = new URL(window.location.href).searchParams.get("routeID");
    if (routeID) {
      for (const gButton of allGroupBtns) {
        if (gButton.endpointID === routeID) {
          gButton.open();
          return;
        }
      }
    }
  });
});

customElements.define("cst-groupbtn", GroupBtn, { extends: "button" });

class CollapsibleBtn extends HTMLButtonElement {
  open() {
    this.container.classList.remove("hidden");
    this.chevron.style.transform = "rotate(90deg)";
  }

  close() {
    this.container.classList.add("hidden");
    this.chevron.style.transform = "";
  }

  connectedCallback() {
    const id = this.dataset.id;
    if (id == null) {
      return;
    }

    this.chevron = this.querySelector("svg");
    this.chevron.style.transition = "transform .1s";

    const allcontainers = document.querySelectorAll(".collapsible-content");
    this.container = Array.from(allcontainers).find((c) => {
      const containerID = c.dataset.id;
      return containerID == id;
    });

    if (this.container == null) {
      return;
    }

    this.container.__collapsibleContainer = true;
    this.container.__collapsibleBtn = this;

    let isopen = !this.container.classList.contains("hidden");
    this.addEventListener("click", () => {
      if (isopen) {
        this.close();
      } else {
        this.open();
      }
      isopen = !isopen;
    });
  }
}

customElements.define("cst-collapsiblebtn", CollapsibleBtn, { extends: "button" });

/**
 * @typedef {{
 *    element: HTMLElement;
 *    keywords: string[];
 *    children: SearchableNode[];
 * }} SearchableNode
 */

class SearchBar extends HTMLInputElement {
  /** @type SearchableNode[] */
  searchableNodes = [];

  connectedCallback() {
    this.addEventListener("input", () => {
      this.showAll();

      const value = this.value.trim().toLowerCase();
      if (value != "") {
        this.hideNotMatched(this.searchableNodes, value);
      }
    });

    setTimeout(() => {
      this.collectSearchableNodes();
    });
  }

  showAll(nodes = this.searchableNodes) {
    for (const node of nodes) {
      node.element.classList.remove("hidden");
      this.showAll(node.children);
    }
  }

  /**
   * @param {SearchableNode[]} nodes
   * @param {string} svalue
   */
  hideNotMatched(nodes, svalue) {
    for (const node of nodes) {
      if (!this.nodeMatchesSearch(node, svalue)) {
        node.element.classList.add("hidden");
      }
      this.hideNotMatched(node.children, svalue);
    }
  }

  /**
   * @param {SearchableNode} node
   * @param {string} svalue
   */
  nodeMatchesSearch(node, svalue) {
    for (const keyword of node.keywords) {
      if (keyword.includes(svalue)) {
        return true;
      }
    }

    for (const child of node.children) {
      if (this.nodeMatchesSearch(child, svalue)) {
        return true;
      }
    }

    return false;
  }

  /**
   * @param {HTMLElement} node
   */
  getKeywords(node) {
    const name = node.dataset.name;
    const path = node.dataset.path;
    const method = node.dataset.method;

    return [...name.split(" "), ...path.split(" "), ...method.split(" ")].filter(Boolean).map((s) => s.toLowerCase());
  }

  collectSearchableNodes() {
    const searchID = this.getAttribute("searchid");
    const searchElemContainer = document.getElementById(searchID);

    /**
     *
     * @param {SearchableNode[]} acc
     * @param {Element} container
     */
    const getSNodes = (acc, container) => {
      for (const child of container.children) {
        if (child.tagName === "SVG") continue;

        if (child.hasAttribute("snode")) {
          /** @type {SearchableNode} */
          const snode = {
            element: child,
            children: [],
            keywords: this.getKeywords(child),
          };

          acc.push(snode);

          getSNodes(snode.children, child);
        } else {
          getSNodes(acc, child);
        }
      }
    };

    getSNodes(this.searchableNodes, searchElemContainer);
  }
}

customElements.define("search-bar", SearchBar, { extends: "input" });
