const data = {
    name: "top", children: [
        {name: "I", children: [
            {name: "I.a"},  {name: "I.b"}, {name: "I.c"}]},
        {name: "II", children: [
            {name: "II.a"}, {name: "II.b"}]},
        {name: "III", children: [
            {name: "III.a"},  {name: "III.b"}, {name: "III.c"}, {name: "III.d"}]}
    ]
};

const width  = 1000;
const heigth = 800;
const colors = d3.scaleOrdinal(d3.schemeCategory10);

const svg = d3.select("body")
    .append("svg")
    .attr("width", width)
    .attr("height", heigth);

const force = d3.forceSimulation()
    .force("charge", d3.forceManyBody())
    .force("x", d3.forceX(width / 2))
    .force("y", d3.forceY(heigth / 2));

// at first, add the top node, and its children by using expand()
const nodes = [];
const links = [];
nodes.push(data);
expand(data);
setup();

function setup() {
    force.nodes(nodes); 
    force.force("link",
        d3.forceLink(links).strength(1).distance(100));

    // SOLUTION: do not use variables for the links
    svg.selectAll("line.link")
          .data(links)
        .enter().insert("line")
          // SOLUTION: add the class attribute
          .attr("class", 'link')
          .style("stroke", "#999")
          .style("stroke-width", "1px");

    // SOLUTION: do not use variables for the nodes
    svg.selectAll("circle.node")
          .data(nodes)
        .enter().append("circle")
          // SOLUTION: add the class attribute
          .attr("class", "node")
          .attr("r", 4.5)
          .style("fill", function(d) {
              return colors(d.parent && d.parent.name);
          })
          .style("stroke", "#000")
          .on("click", function(datum) {
              force.stop();
              expand(datum);
              setup();
              // SOLUTION: reset alpha, for the simulation to actually run again
              if ( force.alpha() < 0.05 ) {
                  force.alpha(0.05);
              }
              force.restart();
          });

    force.on("tick", function(e) {
        // SOLUTION: do not use variables for the links and nodes
        svg.selectAll("line.link")
            .attr("x1", function(d) { return d.source.x; }) 
            .attr("y1", function(d) { return d.source.y; }) 
            .attr("x2", function(d) { return d.target.x; }) 
            .attr("y2", function(d) { return d.target.y; }); 
        svg.selectAll("circle.node")
            .attr("cx", function(d) { return d.x; }) 
            .attr("cy", function(d) { return d.y; }); 
    });
}

function expand(node) {
    console.log(`Expand ${node.name}, expanded: ${node.expanded}`);
    if ( ! node.expanded ) {
        (node.children || []).forEach(function(child) {
            console.log(`  - child ${child.name}`);
            child.parent = node;
            // pop up around the "parent" node
            child.x = node.x;
            child.y = node.y;
            // add the node, and its link to the "parent"
            nodes.push(child);
            links.push({ source: node, target: child });
        });
        node.expanded = true;
    }
}