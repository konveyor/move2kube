import re
import ipdb
import json
import time
import argparse
from os.path import join, isfile
from os import listdir
from collections import defaultdict
from pprint import pprint

import bashlex
#from dockerfile_parse import DockerfileParser

from gensim.test.utils import datapath
from gensim import utils
import gensim.models

from sklearn.feature_extraction.text import CountVectorizer
from sklearn.decomposition import IncrementalPCA   
from sklearn.manifold import TSNE
from sklearn.cluster import KMeans
import numpy as np

def get_files_in_folder(path):
    return [f for f in listdir(path) if isfile(join(path, f))]

def get_parsed_executions():

    with open("/Users/pablo/Desktop/image_processor_output_experimental/all_executions.json", "r") as f:
        all_executions = json.load(f)

    all_executions_parsed = []
    for ex in all_executions:
        try: 
            ex = ex.replace("#(nop)", "")
            # https://forums.docker.com/t/what-is-nop-and-where-does-it-come-from/32665
            parsed = list(bashlex.split(ex))

            # custom parsing 
            parsed_new = []
            for token in parsed: 
                if "=" in token: 
                    parsed_new.extend(token.split("="))
                else:
                    parsed_new.append(token)
                    

            all_executions_parsed.append({"original": ex , "parsed": parsed_new})
        except:
            #print("error", ex)
            #print("---")
            pass
    with open("/Users/pablo/Desktop/image_processor_output_experimental/all_executions_parsed.json", "w") as f:
        json.dump(all_executions_parsed, f)

    return all_executions_parsed

def get_executions():

    images = {
    "wildfly":  "/Users/pablo/Desktop/image_processor_output/jboss_sep_wildfly",
    "liberty":  "/Users/pablo/Desktop/image_processor_output/openliberty_sep_open-liberty"    
    }

    all_executions = []

    for image, image_path in images.items(): 
        image_list  = get_files_in_folder(image_path)
        for im in image_list:
            print(" ", im)
            try: 
                with open(join(image_path, im), "r") as f:
                    im_data = json.load(f)

                if "history" in im_data:
                    im_data_history = im_data["history"]
                    for h in im_data_history:
                        if "created_by" in h:

                            all_executions.append(h["created_by"])
            except:
                print("could not open file")

    with open("/Users/pablo/Desktop/image_processor_output_experimental/all_executions.json", "w") as f:
        json.dump(all_executions, f)

    return all_executions


def explore():

    all_executions_parsed = get_parsed_executions()
    parsed_list = [i["parsed"] for i in all_executions_parsed]
    vectorizer = CountVectorizer(tokenizer=lambda doc: doc, lowercase=False,analyzer="word")
    cv_fit = vectorizer.fit_transform(parsed_list)

    test = list(zip( vectorizer.get_feature_names(), cv_fit.toarray().sum(axis=0)))

    test1 = sorted(test, key= lambda x : x[1], reverse=True)
    test2 = sorted(test) 
    ipdb.set_trace()


class ExecCorpus:
    """An iterator that yields sentences (lists of str)."""

    def __iter__(self):
        parsed_executions = get_parsed_executions()
        for pe in parsed_executions:
            # assume there's one document per line, tokens separated by whitespace
            yield pe["parsed"] #utils.simple_preprocess(line)


def reduce_dimensions(model):
    num_dimensions = 2

    # extract the words & their vectors, as numpy arrays
    vectors = np.asarray(model.wv.vectors)
    labels = np.asarray(model.wv.index_to_key)  # fixed-width numpy strings

    kmeans = KMeans(n_clusters=20, random_state=0).fit(vectors)
    clusters = kmeans.labels_

    # reduce using t-SNE
    tsne = TSNE(n_components=num_dimensions, random_state=0)
    vectors = tsne.fit_transform(vectors)

    x_vals = [v[0] for v in vectors]
    y_vals = [v[1] for v in vectors]
    return x_vals, y_vals, labels, clusters

def plot_with_matplotlib(x_vals, y_vals, labels, clusters):
    import matplotlib.pyplot as plt
    import random

    random.seed(0)

    plt.figure(figsize=(12, 12))
    plt.scatter(x_vals, y_vals, c=clusters)

    #
    # Label randomly subsampled 25 data points
    #
    #indices = list(range(len(labels)))
    #selected_indices = random.sample(indices, 25)
    #for i in selected_indices:
    #    plt.annotate(labels[i], (x_vals[i], y_vals[i]))

    plt.savefig("/Users/pablo/Desktop/image_processor_output_experimental/visualization.png")


def main():

    """
    print("get executions")
    get_executions()
    print("get executions parsed")
    get_parsed_executions()
    """


    print("embedding")
    sentences = ExecCorpus()
    model = gensim.models.Word2Vec(
        min_count=2,
        window=6
    )

    print("embedding - training")
    model.build_vocab(sentences)
    model.train(sentences, 
        total_examples=model.corpus_count, 
        epochs=model.epochs)
    
    print("reducing dims and clustering")
    x_vals, y_vals, labels, clusters = reduce_dimensions(model)

    print("plotting")
    plot_with_matplotlib(x_vals, y_vals, labels, clusters)

    cluster2items = defaultdict(list)
    for i,c in zip(labels, clusters):
        cluster2items[str(c)].append(i)

    with open("/Users/pablo/Desktop/image_processor_output_experimental/cluster2items.json", "w") as f:
        json.dump(cluster2items, f)


if __name__ == "__main__":
    main()