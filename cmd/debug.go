package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tomohiro-owada/affine-cli/internal/validate"
)

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.Flags().String("doc-id", "", "Document ID (required)")
	_ = debugCmd.MarkFlagRequired("doc-id")
}

var debugCmd = &cobra.Command{
	Use:    "debug-dump",
	Short:  "Dump raw Y.js block structure (debug)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		docID, _ := cmd.Flags().GetString("doc-id")
		if err := validate.DocID(docID); err != nil {
			return err
		}
		sess, err := connectDocOps(cmd)
		if err != nil {
			return err
		}
		defer sess.Close()

		engDocID, err := sess.LoadDoc(docID)
		if err != nil {
			return err
		}

		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var result = {};
				blocks.forEach(function(block, blockId) {
					if (!(block instanceof Y.Map)) return;
					var b = {};
					block.forEach(function(v, k) {
						if (v instanceof Y.Text) {
							b[k] = "__YText:" + v.toString();
						} else if (v instanceof Y.Array) {
							var arr = [];
							v.forEach(function(item) {
								if (item instanceof Y.Map) {
									var obj = {};
									item.forEach(function(val, key) {
										if (val instanceof Y.Text) obj[key] = "__YText:" + val.toString();
										else if (val instanceof Y.Array) {
											var a2 = []; val.forEach(function(i2) { a2.push(i2); }); obj[key] = a2;
										} else if (val instanceof Y.Map) {
											var m2 = {}; val.forEach(function(v2, k2) {
												if (v2 instanceof Y.Text) m2[k2] = "__YText:" + v2.toString();
												else m2[k2] = v2;
											}); obj[key] = m2;
										} else obj[key] = val;
									});
									arr.push(obj);
								} else {
									arr.push(item);
								}
							});
							b[k] = arr;
						} else if (v instanceof Y.Map) {
							var obj = {};
							v.forEach(function(val, key) {
								if (val instanceof Y.Map) {
									var inner = {};
									val.forEach(function(v2, k2) {
										if (v2 instanceof Y.Map) {
											var m3 = {}; v2.forEach(function(v3, k3) {
												if (v3 instanceof Y.Text) m3[k3] = "__YText:" + v3.toString();
												else m3[k3] = v3;
											}); inner[k2] = m3;
										} else if (v2 instanceof Y.Text) {
											inner[k2] = "__YText:" + v2.toString();
										} else inner[k2] = v2;
									});
									obj[key] = inner;
								} else if (val instanceof Y.Text) {
									obj[key] = "__YText:" + val.toString();
								} else {
									obj[key] = val;
								}
							});
							b[k] = {"__YMap": obj};
						} else {
							b[k] = v;
						}
					});
					result[blockId] = b;
				});
				return JSON.stringify(result, null, 2);
			})()
		`, engDocID)

		val, err := sess.Engine.RunScript(script)
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	},
}
